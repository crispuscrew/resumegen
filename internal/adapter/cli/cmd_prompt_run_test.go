package cli

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/crispuscrew/resumegen/internal/adapter/trackrepo"
	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

func ptr(s string) *string { return &s }

func TestResolve_Flag(t *testing.T) {
	rc := resolveCtx{flagVals: map[string]*string{"role": ptr("Staff Eng")}}
	spec := prompt.InputSpec{Source: prompt.SourceFlag, Flag: "role", Required: true}
	got, err := rc.resolve(context.Background(), "role", spec)
	if err != nil || got != "Staff Eng" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestResolve_FlagDefaultAndRequired(t *testing.T) {
	rc := resolveCtx{flagVals: map[string]*string{"tone": ptr("")}}

	def, err := rc.resolve(context.Background(), "tone",
		prompt.InputSpec{Source: prompt.SourceFlag, Flag: "tone", Default: "warm"})
	if err != nil || def != "warm" {
		t.Errorf("default: got %q err %v", def, err)
	}

	_, err = rc.resolve(context.Background(), "tone",
		prompt.InputSpec{Source: prompt.SourceFlag, Flag: "tone", Required: true})
	if err == nil {
		t.Error("required unset flag should error")
	}
}

func TestResolve_JDFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "jd.txt")
	if err := os.WriteFile(f, []byte("hire me"), 0o644); err != nil {
		t.Fatal(err)
	}
	rc := resolveCtx{flagVals: map[string]*string{"jd": ptr(f)}}
	got, err := rc.resolve(context.Background(), "jd",
		prompt.InputSpec{Source: prompt.SourceJDFile, Flag: "jd", Required: true})
	if err != nil || got != "hire me" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestResolve_JDFile_MissingRequired(t *testing.T) {
	rc := resolveCtx{flagVals: map[string]*string{"jd": ptr("")}}
	_, err := rc.resolve(context.Background(), "jd",
		prompt.InputSpec{Source: prompt.SourceJDFile, Flag: "jd", Required: true})
	if err == nil {
		t.Error("missing required jd-file path should error")
	}
}

func TestResolve_Stdin(t *testing.T) {
	rc := resolveCtx{reader: bufio.NewReader(strings.NewReader("piped message\n"))}
	got, err := rc.resolve(context.Background(), "message",
		prompt.InputSpec{Source: prompt.SourceStdin, Required: true})
	if err != nil || got != "piped message" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestResolve_PromptNoInput(t *testing.T) {
	// with a default, --no-input yields the default
	rc := resolveCtx{noInput: true}
	got, err := rc.resolve(context.Background(), "intent",
		prompt.InputSpec{Source: prompt.SourcePrompt, Default: "interested"})
	if err != nil || got != "interested" {
		t.Errorf("got %q err %v", got, err)
	}
	// required with no default -> error, never blocks
	_, err = rc.resolve(context.Background(), "context",
		prompt.InputSpec{Source: prompt.SourcePrompt, Required: true})
	if err == nil {
		t.Error("required interactive input under --no-input should error")
	}
}

func TestResolve_DataDump(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "output"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "output", "default.md"), []byte("# Ada"), 0o644); err != nil {
		t.Fatal(err)
	}
	rc := resolveCtx{appdir: dir, profile: "default"}
	got, err := rc.resolve(context.Background(), "resume",
		prompt.InputSpec{Source: prompt.SourceDataDump, Required: true})
	if err != nil || got != "# Ada" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestResolve_DataDump_MissingRequired(t *testing.T) {
	rc := resolveCtx{appdir: t.TempDir(), profile: "default"}
	_, err := rc.resolve(context.Background(), "resume",
		prompt.InputSpec{Source: prompt.SourceDataDump, Required: true})
	if err == nil || !strings.Contains(err.Error(), "emit_markdown") {
		t.Errorf("want emit_markdown hint, got %v", err)
	}
}

func TestResolve_StdinNoInputTerminalDoesNotBlock(t *testing.T) {
	// A non-*os.File reader is treated as "not a terminal", so to exercise the
	// terminal path we point stdin at a real char device when available.
	dev, err := os.Open(os.DevNull)
	if err != nil {
		t.Skip("no /dev/null")
	}
	defer func() { _ = dev.Close() }()
	// /dev/null is not a char-device TTY, so this asserts the non-blocking read
	// path: an empty, non-required stdin resolves to "" without hanging.
	rc := resolveCtx{noInput: true, stdin: dev, reader: bufio.NewReader(dev)}
	got, err := rc.resolve(context.Background(), "message",
		prompt.InputSpec{Source: prompt.SourceStdin})
	if err != nil || got != "" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestPromptRun_ReservedFlagRejected(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "prompts"), 0o755); err != nil {
		t.Fatal(err)
	}
	tpl := `+++
name = "collide"
description = "x"
[inputs.profile]
source = "flag"
flag = "profile"
required = true
+++
Hi {{profile}}`
	if err := os.WriteFile(filepath.Join(dir, "prompts", "collide.md"), []byte(tpl), 0o644); err != nil {
		t.Fatal(err)
	}
	// Appdir shadows embedded, so an empty skeleton is enough here.
	deps := Deps{Skeleton: fstest.MapFS{}}
	err := promptRun(context.Background(), deps, []string{"collide", "--path", dir, "--profile", "x"})
	if err == nil || !strings.Contains(err.Error(), "reserved flag") {
		t.Fatalf("want reserved-flag error, got %v", err)
	}
}

func TestResolve_AppID(t *testing.T) {
	dir := t.TempDir()
	jdPath := filepath.Join(dir, "jd.txt")
	if err := os.WriteFile(jdPath, []byte("we need Go"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := domain.Application{
		ID: "2026-07-08_acme_go", Company: "Acme", Role: "Go",
		JDPath: jdPath, Status: domain.StatusApplied,
		AppliedAt: time.Date(2026, time.July, 8, 0, 0, 0, 0, time.UTC),
	}
	if err := trackrepo.New(dir).Save(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	rc := resolveCtx{appdir: dir, appID: app.ID}

	// a plain field
	got, err := rc.resolve(context.Background(), "co",
		prompt.InputSpec{Source: prompt.SourceAppID, Field: "company", Required: true})
	if err != nil || got != "Acme" {
		t.Fatalf("company: got %q err %v", got, err)
	}
	// field=jd reads the linked file
	got, err = rc.resolve(context.Background(), "jd",
		prompt.InputSpec{Source: prompt.SourceAppID, Field: "jd", Required: true})
	if err != nil || got != "we need Go" {
		t.Fatalf("jd: got %q err %v", got, err)
	}
	// no --app given -> unset (required errors)
	rc2 := resolveCtx{appdir: dir}
	if _, err := rc2.resolve(context.Background(), "co",
		prompt.InputSpec{Source: prompt.SourceAppID, Field: "company", Required: true}); err == nil {
		t.Error("required app-id input with no --app should error")
	}
}

func TestScanFlag(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--path", "/x"}, "/x"},
		{[]string{"--path=/y"}, "/y"},
		{[]string{"-path", "/z"}, "/z"},
		{[]string{"--other", "v"}, ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got := scanFlag(c.args, "path"); got != c.want {
			t.Errorf("scanFlag(%v) = %q, want %q", c.args, got, c.want)
		}
	}
}

func TestResolve_AppFallbacks(t *testing.T) {
	dir := t.TempDir()
	jdPath := filepath.Join(dir, "jd.txt")
	if err := os.WriteFile(jdPath, []byte("we need Rust"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := domain.Application{
		ID: "2026-07-22_beta_rust", Company: "Beta LLC", Role: "Rust Dev",
		JDPath: jdPath, Status: domain.StatusApplied,
		AppliedAt: time.Date(2026, time.July, 22, 0, 0, 0, 0, time.UTC),
	}
	if err := trackrepo.New(dir).Save(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	rc := resolveCtx{appdir: dir, appID: app.ID, flagVals: map[string]*string{
		"company": ptr(""), "jd": ptr(""),
	}}

	// empty --company falls back to the app's company
	got, err := rc.resolve(context.Background(), "company",
		prompt.InputSpec{Source: prompt.SourceFlag, Flag: "company", Required: true})
	if err != nil || got != "Beta LLC" {
		t.Fatalf("company fallback: got %q err %v", got, err)
	}
	// explicit flag still wins
	rc.flagVals["company"] = ptr("Override Inc")
	got, _ = rc.resolve(context.Background(), "company",
		prompt.InputSpec{Source: prompt.SourceFlag, Flag: "company"})
	if got != "Override Inc" {
		t.Fatalf("explicit flag must win, got %q", got)
	}
	// empty jd-file input reads the app's jd_path
	got, err = rc.resolve(context.Background(), "jd",
		prompt.InputSpec{Source: prompt.SourceJDFile, Flag: "jd", Required: true})
	if err != nil || got != "we need Rust" {
		t.Fatalf("jd fallback: got %q err %v", got, err)
	}
	// TUI path: resolveFromValues honors the same fallbacks
	tpl := prompt.PromptTemplate{Name: "x", Inputs: map[string]prompt.InputSpec{
		"company": {Source: prompt.SourceFlag, Flag: "company", Required: true},
		"jd":      {Source: prompt.SourceJDFile, Flag: "jd", Required: true},
	}, Body: "{{company}} {{jd}}"}
	in, err := rc.resolveFromValues(context.Background(), tpl, map[string]string{})
	if err != nil || in["company"] != "Beta LLC" || in["jd"] != "we need Rust" {
		t.Fatalf("tui fallbacks: %v err %v", in, err)
	}
	// no --app: nothing changes, required empty errors
	rc2 := resolveCtx{appdir: dir, flagVals: map[string]*string{"company": ptr("")}}
	if _, err := rc2.resolve(context.Background(), "company",
		prompt.InputSpec{Source: prompt.SourceFlag, Flag: "company", Required: true}); err == nil {
		t.Error("without --app a required empty flag must still error")
	}
}
