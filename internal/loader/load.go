package loader

import (
	"github.com/crispuscrew/resumegen/internal/model"

	"io/fs"

	toml "github.com/pelletier/go-toml/v2"
)

func loadConfig(fsys fs.FS, path string) (model.Config, error) {
	return load[model.Config](fsys, path)
}

func loadProfile(fsys fs.FS, path string) (model.Profile, error) {
	return load[model.Profile](fsys, path)
}

type jobsFile     struct { Jobs       []model.Job          	`toml:"jobs"` 		}
type projectsFile struct { Projects   []model.Project		`toml:"projects"` 	}
type eduFile      struct { Edu        []model.Edu			`toml:"edu"` 		}
type skillsFile   struct { Categories []model.SkillCat		`toml:"categories"` }

func loadData(fsys fs.FS, dataDir string) (model.ResumeData, error) {
	var data model.ResumeData
	header, err 	:= load[model.Header](	fsys, dataDir + "/header.toml")
	if err != nil {return model.ResumeData{}, err}
	data.Header = header

	jobs, err 		:= load[jobsFile](		fsys, dataDir + "/jobs.toml")
	if err != nil {return model.ResumeData{}, err}
	data.Jobs = jobs.Jobs

	projects, err 	:= load[projectsFile](	fsys, dataDir + "/projects.toml")
	if err != nil {return model.ResumeData{}, err}
	data.Projects = projects.Projects

	education, err 	:= load[eduFile](		fsys, dataDir + "/education.toml")
	if err != nil {return model.ResumeData{}, err}
	data.Edu = education.Edu

	skills, err 	:= load[skillsFile](	fsys, dataDir + "/skills.toml")
	if err != nil {return model.ResumeData{}, err}
	data.SkillCats = skills.Categories
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