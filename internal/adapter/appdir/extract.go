package appdir

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/crispuscrew/resumegen/internal/usecase"
)

// NewSkeletonExtractor returns a usecase.SkeletonExtractor that reads from
// src (typically the binary's embedded defaults) and writes to the host
// filesystem.
func NewSkeletonExtractor(src fs.FS) usecase.SkeletonExtractor {
	return skeletonExtractor{src: src}
}

type skeletonExtractor struct {
	src fs.FS
}

func (s skeletonExtractor) ListSubtree(_ context.Context, subtree string) ([]string, error) {
	info, err := fs.Stat(s.src, subtree)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("skeleton subtree %q is not a directory", subtree)
	}
	var paths []string
	err = fs.WalkDir(s.src, subtree, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			paths = append(paths, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func (s skeletonExtractor) ExtractFile(_ context.Context, srcPath, dst string) (bool, error) {
	if _, err := os.Stat(dst); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("stat %s: %w", dst, err)
	}
	raw, err := fs.ReadFile(s.src, srcPath)
	if err != nil {
		return false, fmt.Errorf("read skeleton %s: %w", srcPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), dirPerm); err != nil {
		return false, fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, raw, filePerm); err != nil {
		return false, fmt.Errorf("write %s: %w", dst, err)
	}
	return true, nil
}
