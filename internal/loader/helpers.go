package loader

import (
	"github.com/crispuscrew/resumegen"

	"os"
	"io/fs"
	"fmt"
	"path/filepath"
	"strings"
	"errors"
)

const (
	dirPerm  fs.FileMode = 0o755 // rwxr-xr-x                                          
    filePerm fs.FileMode = 0o644 // rw-r--r--
)

func copyDefaultAppDir(appDirPath string, userChoise func(msg string, defaultVal bool) (bool)) error {
	sub, err := fs.Sub(resumegen.Defaults, "defaultAppDir")
	if err != nil { return err }

	return fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil { return err }

		dst := filepath.Join(appDirPath, path)
		if d.IsDir() { return os.MkdirAll(dst, dirPerm) }
		if _, err := os.Stat(dst); err == nil {
			if !userChoise(fmt.Sprintf("File %s already exists. Do you want to overwrite it?", dst), false) {
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

var errRerun = errors.New("Rerun")
func appDirSmthNotFound(what string, appDirPath string, userChoise func(msg string, defaultVal bool) (bool)) error {
	if userChoise(fmt.Sprintf("%s not found. Do you want copy defaults AppDir? (Its idempotent)", what), true) {
		err := copyDefaultAppDir(appDirPath, userChoise)
		if err != nil { return fmt.Errorf("failed to copy default appDir: %w", err) }
		return errRerun
	}
	return nil
}