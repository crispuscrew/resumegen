package host_test

import (
	"context"
	"io/fs"
	"os"
	"os/exec"
	"testing"

	"github.com/crispuscrew/resumegen"
	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
	"github.com/crispuscrew/resumegen/internal/adapter/render/host"
	"github.com/crispuscrew/resumegen/internal/adapter/tomlrepo"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

// End-to-end render smoke test. Skipped when typst is unavailable so it does
// not gate CI environments without the binary. Asserts the resulting file is
// a non-trivial PDF and the page count is positive.
func TestRender_DefaultAppdir_Integration(t *testing.T) {
	if _, err := exec.LookPath("typst"); err != nil {
		t.Skip("typst not installed; skipping integration test")
	}

	tmp := t.TempDir()
	skeleton, err := fs.Sub(resumegen.Defaults, "defaultAppDir")
	if err != nil {
		t.Fatalf("sub-fs: %v", err)
	}
	approve := func(string, bool) bool { return true }
	if err := appdir.CopySkeleton(skeleton, tmp, approve); err != nil {
		t.Fatalf("copy skeleton: %v", err)
	}

	fsys := os.DirFS(tmp)
	ctx := context.Background()
	cfg, err := tomlrepo.NewConfigSource(fsys).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := tomlrepo.NewProfileRepo(fsys).Load(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	data, err := tomlrepo.NewResumeRepo(fsys).Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	data = usecase.Score(data, profile.Tags, cfg.Score)

	r := host.Renderer{Appdir: tmp}
	outPath, pages, err := r.Render(ctx, data, profile, cfg)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Size() < 1024 {
		t.Errorf("output PDF unexpectedly small: %d bytes", info.Size())
	}

	head := make([]byte, 5)
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Read(head)
	_ = f.Close()
	if string(head) != "%PDF-" {
		t.Errorf("output is not a PDF: head=%q", head)
	}
	if pages <= 0 {
		t.Errorf("page count must be positive, got %f", pages)
	}
	t.Logf("rendered %s (%.3f pages, %d bytes)", outPath, pages, info.Size())
}
