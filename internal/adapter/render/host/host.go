// Package host implements usecase.Renderer by exec-ing a host typst binary.
package host

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/crispuscrew/resumegen/internal/domain"
)

const (
	dirPerm  fs.FileMode = 0o755
	filePerm fs.FileMode = 0o644
)

// Renderer compiles a resume to PDF using the host's typst binary.
// Concurrent invocations against the same Appdir race on templates/data_gen.typ;
// callers must serialize.
type Renderer struct {
	Appdir string
}

// Render writes the generated typst source into <Appdir>/templates/data_gen.typ,
// invokes `typst compile` to produce the PDF, then `typst query` to determine
// the rendered page count. Returns the absolute PDF path and the page count.
func (r Renderer) Render(ctx context.Context, data domain.ResumeData, profile domain.Profile, cfg domain.Config) (string, float64, error) {
	src, err := BuildTypstSource(data, profile)
	if err != nil {
		return "", 0, fmt.Errorf("build typst source: %w", err)
	}

	dataGenPath := filepath.Join(r.Appdir, "templates", "data_gen.typ")
	if err := os.WriteFile(dataGenPath, src, filePerm); err != nil {
		return "", 0, fmt.Errorf("write data_gen.typ: %w", err)
	}
	defer func() { _ = os.Remove(dataGenPath) }()

	outPath := filepath.Join(r.Appdir, cfg.Paths.OutputDir, profile.Output)
	typPath := filepath.Join(r.Appdir, "templates", "resume.typ")

	if err := os.MkdirAll(filepath.Dir(outPath), dirPerm); err != nil {
		return "", 0, fmt.Errorf("mkdir output: %w", err)
	}

	cmd := exec.CommandContext(ctx, cfg.Paths.TypstBin, "compile", typPath, outPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", 0, fmt.Errorf("typst compile: %w", err)
	}

	pages, err := queryPages(ctx, cfg.Paths.TypstBin, typPath, cfg.Render.PageHeightPt)
	if err != nil {
		return "", 0, fmt.Errorf("typst query: %w", err)
	}
	return outPath, pages, nil
}

func queryPages(ctx context.Context, typstBin, typPath string, pageHeightPt float64) (float64, error) {
	out, err := exec.CommandContext(ctx, typstBin, "query",
		typPath, "<end-marker>", "--field", "value",
	).Output()
	if err != nil {
		return 0, err
	}

	type typstPos struct {
		Page int    `json:"page"`
		X    string `json:"x"`
		Y    string `json:"y"`
	}
	var positions []typstPos
	if err := json.Unmarshal(out, &positions); err != nil {
		return 0, err
	}
	if len(positions) == 0 {
		return 0, errors.New("end-marker not found")
	}

	pos := positions[0]
	y, err := strconv.ParseFloat(strings.TrimSuffix(pos.Y, "pt"), 64)
	if err != nil {
		return 0, err
	}
	return float64(pos.Page-1) + y/pageHeightPt, nil
}
