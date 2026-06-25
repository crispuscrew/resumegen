package appdir_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
)

// Verifies that WalkUp finds a marker placed at the start directory and at
// every ancestor up to the filesystem root, and reports no-match cleanly
// when no marker exists between start and root.
func TestWalkUp(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// No marker anywhere → not found.
	if dir, ok := appdir.WalkUp(deep); ok {
		t.Fatalf("WalkUp before marker: got (%q, true), want (_, false)", dir)
	}

	// Place a marker two levels above deep and confirm it's discovered.
	marker := filepath.Join(root, "a", ".resumegen", "workspace.toml")
	if err := os.MkdirAll(filepath.Dir(marker), 0o755); err != nil {
		t.Fatalf("mkdir marker dir: %v", err)
	}
	if err := os.WriteFile(marker, []byte("schema_version = \"1.1\"\n"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	gotDir, ok := appdir.WalkUp(deep)
	if !ok {
		t.Fatalf("WalkUp after marker: ok=false, want true")
	}
	wantDir := filepath.Join(root, "a")
	if gotDir != wantDir {
		t.Fatalf("WalkUp: got %q, want %q", gotDir, wantDir)
	}

	// Start exactly at the workspace directory → still finds it.
	if dir, ok := appdir.WalkUp(wantDir); !ok || dir != wantDir {
		t.Fatalf("WalkUp at workspace: got (%q, %v), want (%q, true)", dir, ok, wantDir)
	}
}

// Verifies that ResolveActive picks the right source in the three cases:
// explicit --path, walk-up, default fallback.
func TestResolveActive(t *testing.T) {
	root := t.TempDir()
	defDir := filepath.Join(root, "default")
	wsDir := filepath.Join(root, "ws")
	if err := os.MkdirAll(defDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(wsDir, ".resumegen"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wsDir, ".resumegen", "workspace.toml"), []byte("schema_version=\"1.1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("flag", func(t *testing.T) {
		res, err := appdir.ResolveActive(defDir, wsDir, defDir)
		if err != nil {
			t.Fatal(err)
		}
		if res.Source != appdir.SourceFlag {
			t.Fatalf("source: got %v, want SourceFlag", res.Source)
		}
		if res.Dir != defDir {
			t.Fatalf("dir: got %q, want %q", res.Dir, defDir)
		}
	})

	t.Run("walkup", func(t *testing.T) {
		// CWD is somewhere inside the workspace; --path is empty.
		inside := filepath.Join(wsDir, "deep")
		if err := os.MkdirAll(inside, 0o755); err != nil {
			t.Fatal(err)
		}
		res, err := appdir.ResolveActive("", inside, defDir)
		if err != nil {
			t.Fatal(err)
		}
		if res.Source != appdir.SourceWalkUp {
			t.Fatalf("source: got %v, want SourceWalkUp", res.Source)
		}
		if res.Dir != wsDir {
			t.Fatalf("dir: got %q, want %q", res.Dir, wsDir)
		}
		if !res.HasMarker {
			t.Fatalf("HasMarker: got false, want true")
		}
	})

	t.Run("default", func(t *testing.T) {
		// CWD is outside any workspace; --path is empty → fall back to default.
		outside := filepath.Join(root, "elsewhere")
		if err := os.MkdirAll(outside, 0o755); err != nil {
			t.Fatal(err)
		}
		res, err := appdir.ResolveActive("", outside, defDir)
		if err != nil {
			t.Fatal(err)
		}
		if res.Source != appdir.SourceDefault {
			t.Fatalf("source: got %v, want SourceDefault", res.Source)
		}
		if res.Dir != defDir {
			t.Fatalf("dir: got %q, want %q", res.Dir, defDir)
		}
	})
}
