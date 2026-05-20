// Package appdir contains the filesystem workspace adapter: path resolution
// and the Bootstrap port (copy embedded defaults to the target directory on
// first run).
package appdir

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	dirPerm  fs.FileMode = 0o755 // rwxr-xr-x
	filePerm fs.FileMode = 0o644 // rw-r--r--
)

// UserChoice is the prompt callback the Bootstrap adapter uses before any
// destructive action. Returning true approves; false declines.
type UserChoice func(msg string, defaultVal bool) bool

// Resolve expands a leading ~/ and converts the path to an absolute one.
func Resolve(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Abs(path)
}

// Bootstrap copies an embedded skeleton into Target after asking the user.
// Implements usecase.Bootstrap.
type Bootstrap struct {
	Skeleton fs.FS // root of the embedded skeleton
	Target   string
	Choice   UserChoice
}

// EnsureWorkspace prompts the user; if they accept, copies the skeleton.
// Returning a non-nil error means the workspace is not ready and the caller
// should abort.
func (b Bootstrap) EnsureWorkspace(ctx context.Context, hint string) error {
	msg := fmt.Sprintf("%s not found. Do you want copy defaults AppDir? (Its idempotent)", hint)
	if !b.Choice(msg, true) {
		return fmt.Errorf("workspace not initialized: %s missing", hint)
	}
	return CopySkeleton(b.Skeleton, b.Target, b.Choice)
}

// CopySkeleton walks src and copies every file into target, prompting before
// overwriting any pre-existing destination file. Directories are created with
// 0755 and files with 0644.
func CopySkeleton(src fs.FS, target string, choice UserChoice) error {
	return fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dst := filepath.Join(target, p)
		if d.IsDir() {
			return os.MkdirAll(dst, dirPerm)
		}
		if _, err := os.Stat(dst); err == nil {
			if !choice(fmt.Sprintf("File %s already exists. Do you want to overwrite it?", dst), false) {
				return nil
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		raw, err := fs.ReadFile(src, p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, raw, filePerm)
	})
}
