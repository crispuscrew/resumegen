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

	"github.com/crispuscrew/resumegen/internal/adapter/render/sanitize"
	"github.com/crispuscrew/resumegen/internal/domain"
)

const (
	dirPerm fs.FileMode = 0o755
	// data_gen.typ carries the full resume data: private by default.
	filePerm fs.FileMode = 0o600
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
	mode := sanitize.Strict
	if cfg.Render.ForceUnsafe {
		mode = sanitize.Permissive
	}
	src, err := BuildTypstSource(data, profile, mode)
	if err != nil {
		return "", 0, fmt.Errorf("build typst source: %w", err)
	}

	dataGenPath := filepath.Join(r.Appdir, "templates", "data_gen.typ")
	if err := os.WriteFile(dataGenPath, src, filePerm); err != nil {
		return "", 0, fmt.Errorf("write data_gen.typ: %w", err)
	}
	defer func() { _ = os.Remove(dataGenPath) }()

	outPath := filepath.Join(r.Appdir, cfg.Paths.OutputDir, profile.Output)
	if err := EnsureContained(r.Appdir, outPath); err != nil {
		return "", 0, err
	}
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

// EnsureContained reports an error unless outPath resolves to a location
// inside appdir. profile.Output and paths.output_dir are free-form config
// strings, so a "../" escape must not let a render write outside the appdir.
// Both renderers call this: the host one and the container one, which
// additionally bind-mounts the output directory read-write.
//
// Symlinks are resolved before comparing, because filepath.Abs is purely
// lexical: with appdir/output symlinked elsewhere, a lexical check passes
// while the file lands outside. Whoever can write config.toml — the guard's
// own threat model — can also plant that symlink. The deepest existing
// ancestor is resolved, since the output directory usually does not exist yet.
func EnsureContained(appdir, outPath string) error {
	fail := func() error {
		return fmt.Errorf("output path %q escapes the appdir; check profile.output and paths.output_dir", outPath)
	}
	root, err := filepath.EvalSymlinks(appdir)
	if err != nil {
		return fail()
	}
	absOut, err := filepath.Abs(outPath)
	if err != nil {
		return fail()
	}
	// Walk up to the nearest ancestor that exists, resolve it, then re-append
	// the not-yet-created remainder.
	dir, rest := filepath.Dir(absOut), filepath.Base(absOut)
	for {
		resolved, err := filepath.EvalSymlinks(dir)
		if err == nil {
			absOut = filepath.Join(resolved, rest)
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return fail()
		}
		dir, rest = parent, filepath.Join(filepath.Base(dir), rest)
	}
	if !strings.HasPrefix(absOut, filepath.Clean(root)+string(filepath.Separator)) {
		return fail()
	}
	return nil
}

func queryPages(ctx context.Context, typstBin, typPath string, pageHeightPt float64) (float64, error) {
	out, err := exec.CommandContext(ctx, typstBin, "query",
		typPath, "<end-marker>", "--field", "value",
	).Output()
	if err != nil {
		return 0, err
	}
	return ParseQueryPages(out, pageHeightPt)
}

// ParseQueryPages parses the JSON emitted by `typst query <typ> <end-marker>
// --field value` and converts the end-marker's page/y position into a
// fractional page count. Exported so adapters that run typst out-of-process
// (e.g. inside a container) can reuse the same parser.
func ParseQueryPages(raw []byte, pageHeightPt float64) (float64, error) {
	type typstPos struct {
		Page int    `json:"page"`
		X    string `json:"x"`
		Y    string `json:"y"`
	}
	var positions []typstPos
	if err := json.Unmarshal(raw, &positions); err != nil {
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
