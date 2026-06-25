package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/crispuscrew/resumegen/internal/adapter/container"
	"github.com/crispuscrew/resumegen/internal/domain"
)

// noopRunner satisfies container.Runner: every Run/Output succeeds with empty
// data. Used to exercise the router's branching without actually invoking
// podman or typst.
type noopRunner struct {
	queryReply []byte
}

func (n noopRunner) Run(_ context.Context, _ string, _ []string, _ io.Writer, _ io.Writer) error {
	return nil
}
func (n noopRunner) Output(_ context.Context, _ string, _ []string) ([]byte, error) {
	return n.queryReply, nil
}

func newRouter(appdir string, engineOK bool, banner io.Writer) renderRouter {
	return renderRouter{
		appdir:    appdir,
		engine:    container.Engine{Name: "podman", Bin: "/usr/bin/podman"},
		engineOK:  engineOK,
		image:     "localhost/resumegen-render:test",
		cfile:     []byte("FROM scratch\n"),
		runner:    noopRunner{queryReply: []byte(`[{"page":1,"x":"0pt","y":"400pt"}]`)},
		bannerOut: banner,
	}
}

func setupAppdir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "output"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRouter_Auto_FallsBackToHost_WhenNoEngine(t *testing.T) {
	// host path requires a working typst on PATH, so we just confirm the
	// banner mentions host fallback and the host renderer was selected.
	// We do not actually run typst here.
	appdir := setupAppdir(t)
	var banner bytes.Buffer
	r := newRouter(appdir, false /* engineOK */, &banner)

	// Force the host path: in this test we want to confirm that the auto
	// branch with no engine prints the host banner and routes to host. We
	// stop short of actually running typst by passing a config that will
	// cause host.Renderer to fail at compile time — we only assert on the
	// banner content.
	cfg := domain.Config{
		Paths:  domain.Paths{OutputDir: "output", TypstBin: "/nonexistent/typst"},
		Render: domain.Render{PageHeightPt: 842, UseContainer: "auto"},
	}
	_, _, _ = r.Render(context.Background(), domain.ResumeData{}, domain.Profile{Lang: "en", Output: "x.pdf"}, cfg)
	if !strings.Contains(banner.String(), "rendering: host (no container engine on PATH)") {
		t.Errorf("banner = %q", banner.String())
	}
}

func TestRouter_On_RequiresEngine(t *testing.T) {
	appdir := setupAppdir(t)
	r := newRouter(appdir, false, io.Discard)
	cfg := domain.Config{
		Paths:  domain.Paths{OutputDir: "output"},
		Render: domain.Render{PageHeightPt: 842, UseContainer: "true"},
	}
	_, _, err := r.Render(context.Background(), domain.ResumeData{}, domain.Profile{Lang: "en", Output: "x.pdf"}, cfg)
	if err == nil || !strings.Contains(err.Error(), "no container engine") {
		t.Fatalf("err = %v, want one mentioning 'no container engine'", err)
	}
}

func TestRouter_On_BuildsAndRendersInContainer(t *testing.T) {
	appdir := setupAppdir(t)
	var banner bytes.Buffer
	r := newRouter(appdir, true, &banner)
	cfg := domain.Config{
		Paths:  domain.Paths{OutputDir: "output"},
		Render: domain.Render{PageHeightPt: 842, UseContainer: "true"},
	}
	outPath, pages, err := r.Render(context.Background(), domain.ResumeData{}, domain.Profile{Lang: "en", Output: "x.pdf"}, cfg)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.HasSuffix(outPath, "output/x.pdf") {
		t.Errorf("outPath = %q", outPath)
	}
	if pages <= 0 {
		t.Errorf("pages = %v", pages)
	}
	if !strings.Contains(banner.String(), "rendering: container (podman)") {
		t.Errorf("banner = %q", banner.String())
	}
}

func TestRouter_Off_NoBanner(t *testing.T) {
	appdir := setupAppdir(t)
	var banner bytes.Buffer
	r := newRouter(appdir, true, &banner)
	cfg := domain.Config{
		Paths:  domain.Paths{OutputDir: "output", TypstBin: "/nonexistent/typst"},
		Render: domain.Render{PageHeightPt: 842, UseContainer: ""},
	}
	_, _, _ = r.Render(context.Background(), domain.ResumeData{}, domain.Profile{Lang: "en", Output: "x.pdf"}, cfg)
	if banner.Len() != 0 {
		t.Errorf("banner unexpectedly written: %q", banner.String())
	}
}

func TestRouter_InvalidMode(t *testing.T) {
	r := newRouter(t.TempDir(), true, io.Discard)
	cfg := domain.Config{Render: domain.Render{UseContainer: "yes"}}
	_, _, err := r.Render(context.Background(), domain.ResumeData{}, domain.Profile{}, cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
