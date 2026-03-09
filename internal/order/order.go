package order

import (
	"github.com/crispuscrew/resumegen/internal/model"

	"slices"
)

func OrderStage(mdl model.Model) (model.Model, error) {
	mdl.Data.Jobs = scoreAndSortJobs(mdl.Data.Jobs, mdl.Profile.Tags)
	for i := range mdl.Data.Jobs {
		mdl.Data.Jobs[i].Bullets = scoreAndSortBullets(mdl.Data.Jobs[i].Bullets, mdl.Profile.Tags)
	}
	mdl.Data.Jobs = slices.DeleteFunc(mdl.Data.Jobs, func(j model.Job) bool {return len(j.Bullets) == 0})

	mdl.Data.Projects = scoreAndSortProjects(mdl.Data.Projects, mdl.Profile.Tags)
	for i := range mdl.Data.Projects {
		mdl.Data.Projects[i].Bullets = scoreAndSortBullets(mdl.Data.Projects[i].Bullets, mdl.Profile.Tags)
	}
	mdl.Data.Projects = slices.DeleteFunc(mdl.Data.Projects, func(p model.Project) bool {return len(p.Bullets) == 0})

	for i := range mdl.Data.Skills {
		mdl.Data.Skills[i].Items = scoreAndSortSkills(mdl.Data.Skills[i].Items, mdl.Profile.Tags)
	}
	mdl.Data.Skills = slices.DeleteFunc(mdl.Data.Skills, func(c model.SkillCategory) bool {return len(c.Items) == 0})
	return mdl, nil
}