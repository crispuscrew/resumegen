package appdir_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
)

// SkeletonExtractor must list every file under a subtree, copy files that
// don't yet exist, and skip files that do.
func TestSkeletonExtractor(t *testing.T) {
	src := fstest.MapFS{
		"templates/resume.typ":   &fstest.MapFile{Data: []byte("// resume")},
		"templates/template.typ": &fstest.MapFile{Data: []byte("// template")},
		"data/header.toml":       &fstest.MapFile{Data: []byte("name = \"x\"")},
	}
	ext := appdir.NewSkeletonExtractor(src)
	ctx := context.Background()

	files, err := ext.ListSubtree(ctx, "templates")
	if err != nil {
		t.Fatalf("ListSubtree: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("ListSubtree: got %d files, want 2; files=%v", len(files), files)
	}

	dstRoot := t.TempDir()
	dst := filepath.Join(dstRoot, "templates", "resume.typ")
	copied, err := ext.ExtractFile(ctx, "templates/resume.typ", dst)
	if err != nil {
		t.Fatalf("ExtractFile (fresh): %v", err)
	}
	if !copied {
		t.Fatalf("ExtractFile (fresh): copied=false, want true")
	}
	raw, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(raw) != "// resume" {
		t.Fatalf("dst content: got %q, want %q", raw, "// resume")
	}

	// Second extract on the same dst must skip.
	copied, err = ext.ExtractFile(ctx, "templates/resume.typ", dst)
	if err != nil {
		t.Fatalf("ExtractFile (skip): %v", err)
	}
	if copied {
		t.Fatalf("ExtractFile (skip): copied=true, want false (already exists)")
	}
}

// Missing subtree must surface a non-nil error (the CLI checks for
// fs.ErrNotExist to print a friendly message).
func TestSkeletonExtractor_MissingSubtree(t *testing.T) {
	src := fstest.MapFS{"templates/resume.typ": &fstest.MapFile{Data: []byte("x")}}
	_, err := appdir.NewSkeletonExtractor(src).ListSubtree(context.Background(), "prompts")
	if err == nil {
		t.Fatalf("ListSubtree on missing subtree: nil error, want one")
	}
}
