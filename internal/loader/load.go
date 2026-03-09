package loader

import (
	"github.com/crispuscrew/resumegen/internal/types"

	"io/fs"

	toml "github.com/pelletier/go-toml/v2"
)

func loadConfig(fsys fs.FS, path string) (types.Config, error) {
	return load[types.Config](fsys, path)
}

func loadProfile(fsys fs.FS, path string) (types.Profile, error) {
	return load[types.Profile](fsys, path)
}

type jobsFile     struct { Jobs       []types.Job          	`toml:"jobs"` 		}
type projectsFile struct { Projects   []types.Project		`toml:"projects"` 	}
type eduFile      struct { Edu        []types.Edu			`toml:"edu"` 		}
type skillsFile   struct { Categories []types.SkillCategory	`toml:"categories"` }

func loadData(fsys fs.FS, dataDir string) (types.ResumeData, error) {
	var data types.ResumeData
	header, err 	:= load[types.Header](	fsys, dataDir + "/header.toml")
	if err != nil {return types.ResumeData{}, err}
	data.Header = header

	jobs, err 		:= load[jobsFile](		fsys, dataDir + "/experience.toml")
	if err != nil {return types.ResumeData{}, err}
	data.Jobs = jobs.Jobs

	projects, err 	:= load[projectsFile](	fsys, dataDir + "/projects.toml")
	if err != nil {return types.ResumeData{}, err}
	data.Projects = projects.Projects

	education, err 	:= load[eduFile](		fsys, dataDir + "/education.toml")
	if err != nil {return types.ResumeData{}, err}
	data.Education = education.Edu

	skills, err 	:= load[skillsFile](	fsys, dataDir + "/skills.toml")
	if err != nil {return types.ResumeData{}, err}
	data.Skills = skills.Categories
	return data, nil
}

func load[T any](fsys fs.FS, path string) (T, error) {
	var result T

	data, err := fs.ReadFile(fsys, path)
	if err != nil { return result, err }

	err = toml.Unmarshal(data, &result)
	if err != nil { return result, err }

	return result, nil
}