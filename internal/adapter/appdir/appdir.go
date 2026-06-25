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

// ResolutionSource tells callers which rule picked the active workspace
// directory. Useful for `config path`-style diagnostics and for tests.
type ResolutionSource int

const (
	// SourceFlag means the caller passed a non-empty userPath (--path).
	SourceFlag ResolutionSource = iota
	// SourceWalkUp means a `.resumegen/workspace.toml` marker was found by
	// walking up from cwd.
	SourceWalkUp
	// SourceDefault means neither --path nor a marker was found; the global
	// default appdir is used.
	SourceDefault
)

// Resolution is the outcome of ResolveActive: the absolute workspace
// directory and how it was chosen.
type Resolution struct {
	Dir       string
	Source    ResolutionSource
	HasMarker bool
}

// ResolveActive picks the active workspace directory. Order:
//
//  1. userPath if non-empty (treated as the explicit --path flag value)
//  2. walk-up from cwd looking for a `.resumegen/workspace.toml` marker
//  3. defaultPath
//
// All three are run through ExpandAbs so the returned Dir is always
// absolute. The directory may not exist yet in cases (1) and (3); the
// caller's Bootstrap adapter handles creation. In case (2) the directory is
// guaranteed to contain a marker (HasMarker == true). HasMarker may also be
// true for cases (1) and (3) if the chosen directory happens to contain a
// marker — useful for the config overlay decision.
func ResolveActive(userPath, cwd, defaultPath string) (Resolution, error) {
	if userPath != "" {
		abs, err := ExpandAbs(userPath)
		if err != nil {
			return Resolution{}, err
		}
		return Resolution{Dir: abs, Source: SourceFlag, HasMarker: isWorkspace(abs)}, nil
	}
	if cwd != "" {
		absCwd, err := ExpandAbs(cwd)
		if err != nil {
			return Resolution{}, err
		}
		if dir, ok := WalkUp(absCwd); ok {
			return Resolution{Dir: dir, Source: SourceWalkUp, HasMarker: true}, nil
		}
	}
	abs, err := ExpandAbs(defaultPath)
	if err != nil {
		return Resolution{}, err
	}
	return Resolution{Dir: abs, Source: SourceDefault, HasMarker: isWorkspace(abs)}, nil
}

// ExpandAbs expands a leading ~ or ~/ to the user's home directory and then
// converts the result to an absolute path. Used wherever a user-facing path
// is read from a flag or argument.
func ExpandAbs(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = home
	} else if strings.HasPrefix(path, "~/") {
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
