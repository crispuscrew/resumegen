package container

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// recordingRunner captures every Run/Output call and returns canned output.
type recordingRunner struct {
	runs       []call
	outputs    []call
	queryReply []byte
}

func (r *recordingRunner) Run(_ context.Context, bin string, args []string, _ io.Writer, _ io.Writer) error {
	r.runs = append(r.runs, call{bin, append([]string(nil), args...)})
	return nil
}

func (r *recordingRunner) Output(_ context.Context, bin string, args []string) ([]byte, error) {
	r.outputs = append(r.outputs, call{bin, append([]string(nil), args...)})
	return r.queryReply, nil
}

// TestRender_HappyPath verifies the renderer:
//   - writes data_gen.typ to the host appdir
//   - issues `<engine> run ... compile ...` then `<engine> run ... query ...`
//   - parses the query output and returns the right output path
//
// We use a real tempdir for the appdir; the runner is stubbed.
func TestRender_HappyPath(t *testing.T) {
	appdir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(appdir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(appdir, "output"), 0o755); err != nil {
		t.Fatal(err)
	}

	queryReply := []byte(`[{"page":1,"x":"100pt","y":"500pt"}]`)
	runner := &recordingRunner{queryReply: queryReply}

	r := Renderer{
		Engine: Engine{Name: "podman", Bin: "/usr/bin/podman"},
		Runner: runner,
		Image:  "localhost/resumegen-render:test",
		Appdir: appdir,
		UID:    1000, GID: 1000,
	}
	cfg := domain.Config{
		Paths:  domain.Paths{OutputDir: "output"},
		Render: domain.Render{PageHeightPt: 842},
	}
	profile := domain.Profile{Lang: "en", Output: "out.pdf"}
	data := domain.ResumeData{}

	outPath, pages, err := r.Render(context.Background(), data, profile, cfg)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if want := filepath.Join(appdir, "output", "out.pdf"); outPath != want {
		t.Errorf("outPath = %q, want %q", outPath, want)
	}
	// page = (1-1) + 500/842 ≈ 0.594
	if pages <= 0 || pages > 1 {
		t.Errorf("pages = %v, want fractional (0,1)", pages)
	}

	// argv assertions
	if len(runner.runs) != 1 {
		t.Fatalf("Run called %d times, want 1", len(runner.runs))
	}
	if len(runner.outputs) != 1 {
		t.Fatalf("Output called %d times, want 1", len(runner.outputs))
	}
	compileArgs := runner.runs[0].args
	if !containsAll(compileArgs, "--read-only", "--network=none", "--cap-drop=ALL", "--user", "1000:1000", "--userns=keep-id") {
		t.Errorf("compile args missing security flags: %v", compileArgs)
	}
	if !lastArgs(compileArgs, "compile", "/work/templates/resume.typ", "/work/output/out.pdf") {
		t.Errorf("compile tail wrong: %v", compileArgs[len(compileArgs)-3:])
	}
	queryArgs := runner.outputs[0].args
	if !lastArgs(queryArgs, "query", "/work/templates/resume.typ", "<end-marker>", "--field", "value") {
		t.Errorf("query tail wrong: %v", queryArgs)
	}

	// data_gen.typ should be cleaned up
	if _, err := os.Stat(filepath.Join(appdir, "templates", "data_gen.typ")); !os.IsNotExist(err) {
		t.Errorf("data_gen.typ leaked: %v", err)
	}
}

func TestRender_PropagatesCompileError(t *testing.T) {
	appdir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(appdir, "templates"), 0o755)

	r := Renderer{
		Engine: Engine{Name: "podman", Bin: "/x"},
		Runner: errorRunner{},
		Image:  "img",
		Appdir: appdir,
		UID:    1, GID: 1,
	}
	_, _, err := r.Render(context.Background(), domain.ResumeData{},
		domain.Profile{Lang: "en", Output: "x.pdf"},
		domain.Config{Paths: domain.Paths{OutputDir: "output"}, Render: domain.Render{PageHeightPt: 842}})
	if err == nil || !strings.Contains(err.Error(), "compile") {
		t.Fatalf("err = %v, want one wrapping 'compile'", err)
	}
}

type errorRunner struct{}

func (errorRunner) Run(_ context.Context, _ string, _ []string, _ io.Writer, _ io.Writer) error {
	return io.ErrUnexpectedEOF
}
func (errorRunner) Output(_ context.Context, _ string, _ []string) ([]byte, error) {
	return nil, io.ErrUnexpectedEOF
}

func containsAll(args []string, needles ...string) bool {
	set := map[string]bool{}
	for _, a := range args {
		set[a] = true
	}
	for _, n := range needles {
		if !set[n] {
			return false
		}
	}
	return true
}

func lastArgs(args []string, want ...string) bool {
	if len(args) < len(want) {
		return false
	}
	tail := args[len(args)-len(want):]
	for i := range want {
		if tail[i] != want[i] {
			return false
		}
	}
	return true
}

// keep go vet happy that bytes is used in case future tests need it.
var _ = bytes.Buffer{}
