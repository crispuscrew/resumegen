package trim

import (
	"github.com/crispuscrew/resumegen/internal/model"
)

func TrimLowest(data model.ResumeData, minElements model.MinElements) (model.ResumeData, bool) {
	min := minIncluded(model.FlatNested(data))
	if (min == nil) { return data, false }
	
	min.Reason = model.Trimmed
	data = trimEmpty(data, minElements)

	return data, true
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
		if job.Reason != model.Included { continue }
		jobBullets := 0
		for _, bullet := range job.Bullets {
			if bullet.Reason == model.Included { jobBullets++ }
		}
		if jobBullets < minElements.JobBullets { data.Jobs[i].Reason = model.Trimmed }
	}

	for i, project := range data.Projects {
		if project.Reason != model.Included { continue }
		projectBullets := 0
		for _, bullet := range project.Bullets {
			if bullet.Reason == model.Included { projectBullets++ }
		}
		if projectBullets < minElements.ProjectBullets { data.Projects[i].Reason = model.Trimmed }
	}

	for i, skillCat := range data.SkillCats {
		if skillCat.Reason != model.Included { continue }
		skillBullets := 0
		for _, skill := range skillCat.Items {
			if skill.Reason == model.Included { skillBullets++ }
		}
		if skillBullets < minElements.SkillItems { data.SkillCats[i].Reason = model.Trimmed }
	}
	return data
}