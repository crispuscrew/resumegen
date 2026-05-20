package host_test

import (
	"bytes"
	"context"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/crispuscrew/resumegen"
	"github.com/crispuscrew/resumegen/internal/adapter/render/host"
	"github.com/crispuscrew/resumegen/internal/adapter/tomlrepo"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

// -update-golden regenerates testdata/golden/*.typ instead of asserting.
// Run after intentional changes to BuildTypstSource and inspect the diff.
var updateGolden = flag.Bool("update-golden", false, "rewrite golden files instead of comparing")

// Exercises the full pure pipeline (load + score + build) against the embedded
// default appdir and compares the generated typst source to a committed golden.
// This is the byte-equivalence gate for v1.1 — the test must hold for v1.0
// data + default profile.
func TestBuildTypstSource_DefaultAppdir_Golden(t *testing.T) {
	skeleton, err := fs.Sub(resumegen.Defaults, "defaultAppDir")
	if err != nil {
		t.Fatalf("sub-fs: %v", err)
	}

	ctx := context.Background()
	cfg, err := tomlrepo.NewConfigSource(skeleton).Load(ctx)
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	profile, err := tomlrepo.NewProfileRepo(skeleton).Load(ctx, "default")
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	data, err := tomlrepo.NewResumeRepo(skeleton).Load(ctx)
	if err != nil {
		t.Fatalf("data: %v", err)
	}
	data = usecase.Score(data, profile.Tags, cfg.Score)

	got, err := host.BuildTypstSource(data, profile)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	goldenPath := filepath.Join("testdata", "golden", "default.typ")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%d bytes)", goldenPath, len(got))
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if !bytes.Equal(got, want) {
		// Write the actual output next to the golden for easier diffing.
		actualPath := goldenPath + ".actual"
		_ = os.WriteFile(actualPath, got, 0o644)
		t.Fatalf("generated typst source differs from %s.\n  actual written to %s\n  re-run with -update-golden after inspecting the diff", goldenPath, actualPath)
	}
}
