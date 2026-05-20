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

// NewConfigSource returns a usecase.ConfigSource backed by fsys.
func NewConfigSource(fsys fs.FS) usecase.ConfigSource { return configSource{fsys} }

// NewProfileRepo returns a usecase.ProfileRepo backed by fsys. Profiles live
// under profiles/<name>.toml.
func NewProfileRepo(fsys fs.FS) usecase.ProfileRepo { return profileRepo{fsys} }

// NewResumeRepo returns a usecase.ResumeRepo backed by fsys. Data files live
// under data/{header,jobs,projects,education,skills}.toml.
func NewResumeRepo(fsys fs.FS) usecase.ResumeRepo { return resumeRepo{fsys} }

type configSource struct{ fsys fs.FS }

func (c configSource) Load(ctx context.Context) (domain.Config, error) {
	return load[domain.Config](c.fsys, "config.toml")
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
