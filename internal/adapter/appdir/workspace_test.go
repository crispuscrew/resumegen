package appdir_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

// Round-trip: Save then Load should yield the same Workspace.
func TestWorkspaceRepo_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	repo := appdir.NewWorkspaceRepo()
	ctx := context.Background()

	in := domain.Workspace{
		SchemaVersion: "1.1",
		Workspace: domain.WorkspaceMeta{
			Name:        "job-search-2026",
			Description: "Q2 2026 search cycle",
		},
	}
	if err := repo.Save(ctx, dir, in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, err := repo.Load(ctx, dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out != in {
		t.Fatalf("round-trip mismatch:\n got  %+v\n want %+v", out, in)
	}
}

// A second Save must refuse to overwrite an existing marker.
func TestWorkspaceRepo_SaveRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	repo := appdir.NewWorkspaceRepo()
	ctx := context.Background()

	first := domain.Workspace{SchemaVersion: "1.1", Workspace: domain.WorkspaceMeta{Name: "first"}}
	if err := repo.Save(ctx, dir, first); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	second := domain.Workspace{SchemaVersion: "1.1", Workspace: domain.WorkspaceMeta{Name: "second"}}
	err := repo.Save(ctx, dir, second)
	if !errors.Is(err, appdir.ErrWorkspaceExists) {
		t.Fatalf("second Save: got %v, want ErrWorkspaceExists", err)
	}

	// Marker still reflects the first Save.
	out, err := repo.Load(ctx, dir)
	if err != nil {
		t.Fatalf("Load after refused overwrite: %v", err)
	}
	if out.Workspace.Name != "first" {
		t.Fatalf("name after refused overwrite: got %q, want %q", out.Workspace.Name, "first")
	}
}

// Loading from a directory with no marker yields the workspace-missing
// sentinel that the orchestrator already knows how to handle.
func TestWorkspaceRepo_LoadMissingIsSentinel(t *testing.T) {
	dir := t.TempDir()
	_, err := appdir.NewWorkspaceRepo().Load(context.Background(), dir)
	if !errors.Is(err, usecase.ErrWorkspaceMissing) {
		t.Fatalf("Load missing: got %v, want ErrWorkspaceMissing wrap", err)
	}
}

// Sanity: Save creates the .resumegen/ directory if it doesn't exist.
func TestWorkspaceRepo_CreatesMarkerDir(t *testing.T) {
	dir := t.TempDir()
	if err := appdir.NewWorkspaceRepo().Save(context.Background(), dir, domain.Workspace{SchemaVersion: "1.1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, appdir.MarkerSubpath)); err != nil {
		t.Fatalf("marker not written: %v", err)
	}
}
