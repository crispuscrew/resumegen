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

// Bootstrap copies embedded defaults into the workspace on first run.
// hint identifies which file triggered the call (e.g., "config.toml").
// Returning nil means the workspace is now ready and the caller should retry;
// returning an error aborts the orchestration.
type Bootstrap interface {
	EnsureWorkspace(ctx context.Context, hint string) error
}
