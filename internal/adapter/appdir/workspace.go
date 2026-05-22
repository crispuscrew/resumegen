package appdir

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

// ErrWorkspaceExists is returned by WorkspaceRepo.Save when a marker file
// already exists at the target. Init never overwrites; callers translate
// this into a friendly "already a workspace" message.
var ErrWorkspaceExists = errors.New("workspace marker already exists")

// NewWorkspaceRepo returns a usecase.WorkspaceRepo backed by the host
// filesystem.
func NewWorkspaceRepo() usecase.WorkspaceRepo { return workspaceRepo{} }

type workspaceRepo struct{}

func (workspaceRepo) Load(_ context.Context, dir string) (domain.Workspace, error) {
	path := filepath.Join(dir, MarkerSubpath)
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return domain.Workspace{}, fmt.Errorf("%w: %s", usecase.ErrWorkspaceMissing, path)
		}
		return domain.Workspace{}, fmt.Errorf("read workspace marker: %w", err)
	}
	var ws domain.Workspace
	if err := toml.Unmarshal(raw, &ws); err != nil {
		return domain.Workspace{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return ws, nil
}

func (workspaceRepo) Save(_ context.Context, dir string, ws domain.Workspace) error {
	markerDir := filepath.Join(dir, ".resumegen")
	path := filepath.Join(markerDir, "workspace.toml")
	if _, err := os.Stat(path); err == nil {
		return ErrWorkspaceExists
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("stat workspace marker: %w", err)
	}
	if err := os.MkdirAll(markerDir, dirPerm); err != nil {
		return fmt.Errorf("mkdir %s: %w", markerDir, err)
	}
	raw, err := toml.Marshal(ws)
	if err != nil {
		return fmt.Errorf("encode workspace marker: %w", err)
	}
	return os.WriteFile(path, raw, filePerm)
}
