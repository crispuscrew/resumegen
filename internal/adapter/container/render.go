package container

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/crispuscrew/resumegen/internal/adapter/render/host"
	"github.com/crispuscrew/resumegen/internal/adapter/render/sanitize"
	"github.com/crispuscrew/resumegen/internal/domain"
)

const (
	dirPerm  fs.FileMode = 0o755
	filePerm fs.FileMode = 0o644
)

// Renderer implements usecase.Renderer by exec-ing typst inside a container.
// Concurrent invocations against the same Appdir race on templates/data_gen.typ;
// callers must serialize.
type Renderer struct {
	Engine Engine // pre-detected engine (podman or docker)
	Runner Runner // defaults to ExecRunner via the constructor
	Image  string // image tag, e.g. localhost/resumegen-render:1.1.0
	Appdir string // host path mounted at /work (read-only)
	UID    int    // host uid passed via --user
	GID    int    // host gid passed via --user
}

// NewRenderer builds a Renderer with sensible defaults: ExecRunner and the
// current process's uid/gid.
func NewRenderer(eng Engine, image, appdir string) Renderer {
	return Renderer{
		Engine: eng,
		Runner: ExecRunner{},
		Image:  image,
		Appdir: appdir,
		UID:    os.Getuid(),
		GID:    os.Getgid(),
	}
}

// Render generates the Typst source on the host (the container mounts /work
// read-only), then runs `typst compile` and `typst query` inside the
// container. The PDF is written to <Appdir>/<OutputDir>/<profile.Output>.
func (r Renderer) Render(ctx context.Context, data domain.ResumeData, profile domain.Profile, cfg domain.Config) (string, float64, error) {
	mode := sanitize.Strict
	if cfg.Render.ForceUnsafe {
		mode = sanitize.Permissive
	}
	src, err := host.BuildTypstSource(data, profile, mode)
	if err != nil {
		return "", 0, fmt.Errorf("build typst source: %w", err)
	}

	dataGenPath := filepath.Join(r.Appdir, "templates", "data_gen.typ")
	if err := os.WriteFile(dataGenPath, src, filePerm); err != nil {
		return "", 0, fmt.Errorf("write data_gen.typ: %w", err)
	}
	defer func() { _ = os.Remove(dataGenPath) }()

	hostOutDir := filepath.Join(r.Appdir, cfg.Paths.OutputDir)
	if err := os.MkdirAll(hostOutDir, dirPerm); err != nil {
		return "", 0, fmt.Errorf("mkdir output: %w", err)
	}
	hostOutPath := filepath.Join(hostOutDir, profile.Output)
	containerOutPath := "/work/" + cfg.Paths.OutputDir + "/" + profile.Output
	containerTypPath := "/work/templates/resume.typ"

	compileSpec := RunSpec{
		Image: r.Image, AppdirRO: r.Appdir, OutputRW: hostOutDir,
		UID: r.UID, GID: r.GID,
		TypstArgs: []string{"compile", containerTypPath, containerOutPath},
	}
	if err := r.Runner.Run(ctx, r.Engine.Bin, r.Engine.RunArgs(compileSpec), os.Stdout, os.Stderr); err != nil {
		return "", 0, fmt.Errorf("typst compile (container): %w", err)
	}

	querySpec := compileSpec
	querySpec.TypstArgs = []string{"query", containerTypPath, "<end-marker>", "--field", "value"}
	queryOut, err := r.Runner.Output(ctx, r.Engine.Bin, r.Engine.RunArgs(querySpec))
	if err != nil {
		return "", 0, fmt.Errorf("typst query (container): %w", err)
	}
	pages, err := host.ParseQueryPages(queryOut, cfg.Render.PageHeightPt)
	if err != nil {
		return "", 0, fmt.Errorf("parse query output: %w", err)
	}
	return hostOutPath, pages, nil
}
