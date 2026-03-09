package loader

import (
	"github.com/crispuscrew/resumegen"
	"github.com/crispuscrew/resumegen/internal/model"
	"github.com/crispuscrew/resumegen/internal/stage"

	"os"
	"io/fs"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	dirPerm  fs.FileMode = 0o755 // rwxr-xr-x                                          
    filePerm fs.FileMode = 0o644 // rw-r--r--
)

func copyDefaultAppDir(mdl model.Model) error {
	sub, err := fs.Sub(resumegen.Defaults, "defaultAppDir")
	if err != nil { return err }

	return fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }

		dst := filepath.Join(mdl.AppDirPath, path)
		if d.IsDir() { return os.MkdirAll(dst, dirPerm) }
		if _, err := os.Stat(dst); err == nil {
			if !mdl.UserChoise(fmt.Sprintf("File %s already exists. Do you want to overwrite it?", dst), false) {
				return nil
			}
		}

		data, err := fs.ReadFile(sub, path)
		if err != nil { return err }

		return os.WriteFile(dst, data, filePerm)
	})
}

func resolvePath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil { return "", err }
		path = filepath.Join(home, path[1:])
	}
	return filepath.Abs(path)
}

func appDirSmthNotFound(mdl model.Model, what string) (model.Model, error) {
	if mdl.UserChoise(fmt.Sprintf("%s not found. Do you want copy defaults AppDir? (Its idempotent)", what), true) {
		if err := copyDefaultAppDir(mdl); err != nil {
			return mdl, fmt.Errorf("failed to copy default appDir: %w", err)
		}
		return mdl, stage.ErrRerun
	}
	return mdl, fmt.Errorf("%s not found", what)
}