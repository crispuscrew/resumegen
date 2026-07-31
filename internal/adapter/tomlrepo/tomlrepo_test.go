package tomlrepo_test

import (
	"context"
	"errors"
	"testing"
	"testing/fstest"

	"github.com/crispuscrew/resumegen/internal/adapter/tomlrepo"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

const globalConfig = `
[paths]
output_dir = "global-output"
typst_bin = "typst"

[render]
page_limit = 1.0
page_height_pt = 841.89

[render.min_elements]
job_bullets = 1
project_bullets = 1
skill_items = 1
`

const workspaceOverlay = `
[paths]
output_dir = "ws-output"

[render]
page_limit = 2.0
`

// Workspace overlay overrides only the keys it defines; everything else
// falls through to global.
func TestLayeredConfig_WorkspaceWinsPerKey(t *testing.T) {
	global := fstest.MapFS{"config.toml": &fstest.MapFile{Data: []byte(globalConfig)}}
	workspace := fstest.MapFS{"config.toml": &fstest.MapFile{Data: []byte(workspaceOverlay)}}

	cfg, err := tomlrepo.NewLayeredConfigSource(global, workspace).Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Paths.OutputDir != "ws-output" {
		t.Errorf("OutputDir: got %q, want %q (workspace should win)", cfg.Paths.OutputDir, "ws-output")
	}
	if cfg.Paths.TypstBin != "typst" {
		t.Errorf("TypstBin: got %q, want %q (workspace silent → global wins)", cfg.Paths.TypstBin, "typst")
	}
	if cfg.Render.PageLimit != 2.0 {
		t.Errorf("PageLimit: got %v, want 2.0", cfg.Render.PageLimit)
	}
	if cfg.Render.PageHeightPt != 841.89 {
		t.Errorf("PageHeightPt: got %v, want 841.89 (from global)", cfg.Render.PageHeightPt)
	}
	if cfg.Render.MinElements.JobBullets != 1 {
		t.Errorf("MinElements.JobBullets: got %v, want 1 (from global)", cfg.Render.MinElements.JobBullets)
	}
}

// Missing workspace overlay file is a tolerated no-op: global values stay.
func TestLayeredConfig_NoOverlayFile(t *testing.T) {
	global := fstest.MapFS{"config.toml": &fstest.MapFile{Data: []byte(globalConfig)}}
	workspace := fstest.MapFS{} // empty FS — no overlay file present

	cfg, err := tomlrepo.NewLayeredConfigSource(global, workspace).Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Paths.OutputDir != "global-output" {
		t.Errorf("OutputDir: got %q, want %q", cfg.Paths.OutputDir, "global-output")
	}
}

// Both global and workspace missing → workspace-missing sentinel.
func TestLayeredConfig_BothMissing(t *testing.T) {
	_, err := tomlrepo.NewLayeredConfigSource(fstest.MapFS{}, fstest.MapFS{}).Load(context.Background())
	if !errors.Is(err, usecase.ErrWorkspaceMissing) {
		t.Fatalf("Load: got %v, want ErrWorkspaceMissing wrap", err)
	}
}

// Workspace-only mode (no global) still works.
func TestLayeredConfig_WorkspaceOnly(t *testing.T) {
	workspace := fstest.MapFS{"config.toml": &fstest.MapFile{Data: []byte(globalConfig)}}
	cfg, err := tomlrepo.NewLayeredConfigSource(nil, workspace).Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Paths.OutputDir != "global-output" {
		t.Errorf("OutputDir: got %q, want %q", cfg.Paths.OutputDir, "global-output")
	}
}
