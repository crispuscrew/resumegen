// Package tomlrepo implements usecase.ConfigSource, ProfileRepo, and
// ResumeRepo by reading TOML files from an fs.FS. Backing the adapter with an
// fs.FS lets the same code serve the host filesystem (os.DirFS) and embedded
// or in-memory test fixtures.
package tomlrepo

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

// NewConfigSource returns a usecase.ConfigSource that reads config.toml from
// fsys with no overlay. Equivalent to NewLayeredConfigSource(fsys, nil).
func NewConfigSource(fsys fs.FS) usecase.ConfigSource {
	return layeredConfigSource{global: fsys}
}

// NewLayeredConfigSource composes a base config FS with an optional workspace
// overlay FS. When overlay is non-nil, its config.toml is decoded on top of
// the base config; absent keys fall through to the base values. Missing files
// are tolerated as long as at least one FS yields a config.toml — that gives
// the caller fine-grained control over which appdir the marker bootstrap runs
// against.
func NewLayeredConfigSource(base, overlay fs.FS) usecase.ConfigSource {
	return layeredConfigSource{global: base, workspace: overlay}
}

// NewProfileRepo returns a usecase.ProfileRepo backed by fsys. Profiles live
// under profiles/<name>.toml.
func NewProfileRepo(fsys fs.FS) usecase.ProfileRepo { return profileRepo{fsys} }

// NewResumeRepo returns a usecase.ResumeRepo backed by fsys. Data files live
// under data/{header,jobs,projects,education,skills}.toml.
func NewResumeRepo(fsys fs.FS) usecase.ResumeRepo { return resumeRepo{fsys} }

type layeredConfigSource struct {
	global    fs.FS
	workspace fs.FS // nil = no overlay
}

func (l layeredConfigSource) Load(_ context.Context) (domain.Config, error) {
	var cfg domain.Config
	var (
		gotGlobal, gotOverlay bool
	)

	if l.global != nil {
		ok, err := decodeInto(&cfg, l.global, "config.toml")
		if err != nil {
			return domain.Config{}, err
		}
		gotGlobal = ok
	}

	if l.workspace != nil {
		ok, err := decodeInto(&cfg, l.workspace, "config.toml")
		if err != nil {
			return domain.Config{}, err
		}
		gotOverlay = ok
	}

	if !gotGlobal && !gotOverlay {
		return domain.Config{}, fmt.Errorf("%w: config.toml", usecase.ErrWorkspaceMissing)
	}
	return cfg, nil
}

// decodeInto reads path from fsys and unmarshals it on top of dst. Returns
// (true, nil) on success, (false, nil) when the file does not exist, and
// (false, err) for any other failure.
func decodeInto(dst *domain.Config, fsys fs.FS, path string) (bool, error) {
	raw, err := fs.ReadFile(fsys, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	if err := toml.Unmarshal(raw, dst); err != nil {
		return false, fmt.Errorf("parse %s: %w", path, err)
	}
	return true, nil
}

type profileRepo struct{ fsys fs.FS }

func (p profileRepo) Load(ctx context.Context, name string) (domain.Profile, error) {
	return load[domain.Profile](p.fsys, "profiles/"+name+".toml")
}

type resumeRepo struct{ fsys fs.FS }

type jobsFile struct {
	Jobs []domain.Job `toml:"jobs"`
}
type projectsFile struct {
	Projects []domain.Project `toml:"projects"`
}
type eduFile struct {
	Edu []domain.Edu `toml:"edu"`
}
type skillsFile struct {
	Categories []domain.SkillCat `toml:"categories"`
}

func (r resumeRepo) Load(ctx context.Context) (domain.ResumeData, error) {
	var data domain.ResumeData

	header, err := load[domain.Header](r.fsys, "data/header.toml")
	if err != nil {
		return domain.ResumeData{}, err
	}
	data.Header = header

	jobs, err := load[jobsFile](r.fsys, "data/jobs.toml")
	if err != nil {
		return domain.ResumeData{}, err
	}
	data.Jobs = jobs.Jobs

	projects, err := load[projectsFile](r.fsys, "data/projects.toml")
	if err != nil {
		return domain.ResumeData{}, err
	}
	data.Projects = projects.Projects

	edu, err := load[eduFile](r.fsys, "data/education.toml")
	if err != nil {
		return domain.ResumeData{}, err
	}
	data.Edu = edu.Edu

	skills, err := load[skillsFile](r.fsys, "data/skills.toml")
	if err != nil {
		return domain.ResumeData{}, err
	}
	data.SkillCats = skills.Categories

	return data, nil
}

func load[T any](fsys fs.FS, path string) (T, error) {
	var result T

	raw, err := fs.ReadFile(fsys, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return result, fmt.Errorf("%w: %s", usecase.ErrWorkspaceMissing, path)
		}
		return result, fmt.Errorf("read %s: %w", path, err)
	}
	if err := toml.Unmarshal(raw, &result); err != nil {
		return result, fmt.Errorf("parse %s: %w", path, err)
	}
	return result, nil
}
