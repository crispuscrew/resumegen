package host_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/crispuscrew/resumegen/internal/adapter/render/host"
)

// EnsureContained is the single guard both renderers use to keep a render
// inside the appdir. The lexical "../" case was already covered; the symlink
// case was not, and filepath.Abs does not resolve symlinks, so a symlinked
// output directory used to defeat the guard entirely.
func TestEnsureContained(t *testing.T) {
	root := t.TempDir()
	appdir := filepath.Join(root, "appdir")
	outside := filepath.Join(root, "OUTSIDE")
	for _, d := range []string{filepath.Join(appdir, "templates"), outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("legitimate path is accepted", func(t *testing.T) {
		if err := host.EnsureContained(appdir, filepath.Join(appdir, "output", "cv.pdf")); err != nil {
			t.Errorf("EnsureContained rejected an in-appdir path: %v", err)
		}
	})

	t.Run("dotdot escape is rejected", func(t *testing.T) {
		if err := host.EnsureContained(appdir, filepath.Join(appdir, "..", "OUTSIDE", "cv.pdf")); err == nil {
			t.Error("EnsureContained accepted a ../ escape")
		}
	})

	t.Run("symlinked output dir is rejected", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation needs privilege on windows")
		}
		link := filepath.Join(appdir, "output")
		if err := os.Symlink(outside, link); err != nil {
			t.Skipf("cannot create symlink: %v", err)
		}
		defer func() { _ = os.Remove(link) }()
		if err := host.EnsureContained(appdir, filepath.Join(link, "cv.pdf")); err == nil {
			t.Error("EnsureContained accepted a path through a symlink pointing outside the appdir")
		}
	})

	t.Run("absolute path outside is rejected", func(t *testing.T) {
		if err := host.EnsureContained(appdir, filepath.Join(outside, "cv.pdf")); err == nil {
			t.Error("EnsureContained accepted an absolute path outside the appdir")
		}
	})
}
