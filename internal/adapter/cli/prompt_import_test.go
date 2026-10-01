package cli

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crispuscrew/resumegen"
	"github.com/crispuscrew/resumegen/internal/adapter/promptrepo"
	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

func TestPromptRun_ImportResume(test *testing.T) {
	directory := test.TempDir()
	resumePath := filepath.Join(directory, "resume.txt")
	outputPath := filepath.Join(directory, "prompt.md")
	if err := os.WriteFile(resumePath, []byte("Ada wrote Go services in 2024."), 0o600); err != nil {
		test.Fatal(err)
	}
	skeleton, err := fs.Sub(resumegen.Defaults, "defaultAppDir")
	if err != nil {
		test.Fatal(err)
	}
	deps := Deps{Skeleton: skeleton}
	args := []string{"import-resume", "--path", directory, "--resume", resumePath, "--output", outputPath, "--no-input"}
	if err := promptRun(context.Background(), deps, args); err != nil {
		test.Fatal(err)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		test.Fatal(err)
	}
	text := string(raw)
	for _, expected := range []string{"Ada wrote Go services in 2024.", "Languages to emit: en", "derive a vocabulary"} {
		if !strings.Contains(text, expected) {
			test.Errorf("missing %q in rendered import prompt", expected)
		}
	}
	if strings.Contains(text, "{{resume}}") {
		test.Error("resume placeholder was not resolved")
	}
	args = append(args, "--tags", "go,backend", "--langs", "en,ru")
	if err := promptRun(context.Background(), deps, args); err != nil {
		test.Fatal(err)
	}
	raw, err = os.ReadFile(outputPath)
	if err != nil {
		test.Fatal(err)
	}
	if !strings.Contains(string(raw), "Tag vocabulary to use: go,backend") || !strings.Contains(string(raw), "Languages to emit: en,ru") {
		test.Fatal("explicit tags/languages did not override defaults")
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		test.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		test.Errorf("prompt permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestImportResume_FileNeverFallsBackToApplication(test *testing.T) {
	skeleton, err := fs.Sub(resumegen.Defaults, "defaultAppDir")
	if err != nil {
		test.Fatal(err)
	}
	template, err := promptrepo.New(skeleton, nil).Load(context.Background(), "import-resume")
	if err != nil {
		test.Fatal(err)
	}
	resolver := resolveCtx{appdir: test.TempDir(), appID: "nonexistent-application"}
	if _, err := resolver.resolve(context.Background(), "resume", template.Inputs["resume"]); err == nil || !strings.Contains(err.Error(), "--resume") {
		test.Fatalf("missing resume must require --resume, got %v", err)
	}
	if _, err := resolver.resolveFromValues(context.Background(), template, map[string]string{}); err == nil || !strings.Contains(err.Error(), "--resume") {
		test.Fatalf("TUI missing resume must require its own file, got %v", err)
	}
	missingPath := filepath.Join(test.TempDir(), "missing.txt")
	resolver.flagVals = map[string]*string{"resume": &missingPath}
	if _, err := resolver.resolve(context.Background(), "resume", template.Inputs["resume"]); err == nil || !strings.Contains(err.Error(), "read") {
		test.Fatalf("unreadable resume must fail, got %v", err)
	}
	if template.Inputs["resume"].Source != prompt.SourceFile {
		test.Fatal("resume import must use an explicit file source")
	}
	emptyPath := filepath.Join(test.TempDir(), "empty.txt")
	if err := os.WriteFile(emptyPath, nil, 0o600); err != nil {
		test.Fatal(err)
	}
	resolver.flagVals["resume"] = &emptyPath
	if _, err := resolver.resolve(context.Background(), "resume", template.Inputs["resume"]); err == nil {
		test.Fatal("an empty source resume must fail")
	}
	if _, err := resolver.resolveFromValues(context.Background(), template, map[string]string{"resume": emptyPath}); err == nil {
		test.Fatal("an empty TUI source resume must fail")
	}
}
