package trim

import (
	"github.com/crispuscrew/resumegen/internal/model"
)

func TrimLowest(data model.ResumeData, minElements model.MinElements) model.ResumeData {
	min := minIncluded(model.FlatNested(data))
	min.Reason = model.Trimmed

	data = trimEmpty(data, minElements)

	return data
}

func minIncluded(metas []*model.Meta) *model.Meta {
	var min *model.Meta
	for _, meta := range metas {
		if meta.Reason != model.Included { continue }
		if min == nil || meta.Score < min.Score { min = meta}
	}
	return min
}

func trimEmpty(data model.ResumeData, minElements model.MinElements) model.ResumeData {
	for i, job := range data.Jobs {
		jobBullets := 0
		for _, bullet := range job.Bullets {
			if bullet.Reason == model.Included { jobBullets++ }
		}
		if jobBullets < minElements.JobBullets { data.Jobs[i].Reason = model.Trimmed }
	}

	for i, project := range data.Projects {
		projectBullets := 0
		for _, bullet := range project.Bullets {
			if bullet.Reason == model.Included { projectBullets++ }
		}
		if projectBullets < minElements.ProjectBullets { data.Projects[i].Reason = model.Trimmed }
	}

	for i, skillCat := range data.SkillCats {
		skillBullets := 0
		for _, skill := range skillCat.Items {
			if skill.Reason == model.Included { skillBullets++ }
		}
		if skillBullets < minElements.SkillCats { data.SkillCats[i].Reason = model.Trimmed }
	}
	return data
}