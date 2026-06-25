// Package usecase orchestrates the resume-generation pipeline using the
// ports defined in this file. Adapters in internal/adapter/* implement these
// ports; cmd/resumegen wires them together.
package usecase

import (
	"context"
	"errors"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// ErrWorkspaceMissing is the sentinel adapters wrap when a workspace file is
// not present on disk. The orchestrator detects it without importing io/fs.
var ErrWorkspaceMissing = errors.New("workspace file missing")

// ErrCannotFitPage is returned when the trim loop runs out of trimmable
// elements before the rendered page count fits inside the configured limit.
var ErrCannotFitPage = errors.New("cannot trim further; page limit not met")

// ConfigSource loads the workspace's config.toml.
type ConfigSource interface {
	Load(ctx context.Context) (domain.Config, error)
}

// ProfileRepo loads a named profile from profiles/<name>.toml.
type ProfileRepo interface {
	Load(ctx context.Context, name string) (domain.Profile, error)
}

// ResumeRepo loads the workspace's resume data (header, jobs, projects,
// education, skills).
type ResumeRepo interface {
	Load(ctx context.Context) (domain.ResumeData, error)
}

// Renderer compiles a resume to PDF and reports the result's page count.
// Implementations decide where the PDF is written and return its path.
type Renderer interface {
	Render(ctx context.Context, data domain.ResumeData, profile domain.Profile, cfg domain.Config) (outPath string, pages float64, err error)
}

// PDFPostProcessor optionally rewrites a freshly rendered PDF in place (e.g.
// qpdf metadata stripping). The orchestrator invokes it once — after the trim
// loop has settled on the final PDF — and only when enabled by config.
type PDFPostProcessor interface {
	Strip(ctx context.Context, pdfPath string) error
}

// Bootstrap copies embedded defaults into the workspace on first run.
// hint identifies which file triggered the call (e.g., "config.toml").
// Returning nil means the workspace is now ready and the caller should retry;
// returning an error aborts the orchestration.
type Bootstrap interface {
	EnsureWorkspace(ctx context.Context, hint string) error
}

// WorkspaceRepo reads and writes the .resumegen/workspace.toml marker that
// identifies a directory as a resumegen workspace. dir is the workspace root
// (not the marker path); implementations append the marker subpath.
type WorkspaceRepo interface {
	Load(ctx context.Context, dir string) (domain.Workspace, error)
	Save(ctx context.Context, dir string, ws domain.Workspace) error
}

// SkeletonExtractor copies files from an embedded skeleton FS to host paths.
// Used by `init` and the `extract` subcommands. Implementations never
// overwrite existing destination files.
type SkeletonExtractor interface {
	// ListSubtree returns skeleton-relative file paths (e.g.
	// "templates/resume.typ") for every file under subtree. The leading
	// subtree segment is preserved in the returned paths so callers can use
	// them for both source lookup and destination writes.
	ListSubtree(ctx context.Context, subtree string) ([]string, error)
	// ExtractFile copies srcPath (a path inside the skeleton FS) to dst on
	// the host filesystem. Returns (true, nil) on copy, (false, nil) when
	// dst already exists (skip), or a non-nil error.
	ExtractFile(ctx context.Context, srcPath, dst string) (copied bool, err error)
}
