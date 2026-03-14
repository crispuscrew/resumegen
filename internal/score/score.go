package score

import (
	"github.com/crispuscrew/resumegen/internal/model"
)

func Score(data model.ResumeData, profileTags []string, priority model.Score) model.ResumeData {
	data.Jobs = scoreAndFilter[model.Job, *model.Job](data.Jobs, profileTags)
	for i := range data.Jobs {
		data.Jobs[i].Bullets = scoreAndFilter[model.Bullet, *model.Bullet](data.Jobs[i].Bullets, profileTags)
	}

	data.Projects = scoreAndFilter[model.Project, *model.Project](data.Projects, profileTags)
	for i := range data.Projects {
		data.Projects[i].Bullets = scoreAndFilter[model.Bullet, *model.Bullet](data.Projects[i].Bullets, profileTags)
	}

	data.SkillCats = scoreAndFilter[model.SkillCat, *model.SkillCat](data.SkillCats, profileTags)
	for i := range data.SkillCats {
		data.SkillCats[i].Items = scoreAndFilter[model.SkillItem, *model.SkillItem](data.SkillCats[i].Items, profileTags)
		data.SkillCats[i].Items = addPriority[model.SkillItem, *model.SkillItem](data.SkillCats[i].Items, priority.SkillPriority)
	}
	return data
}

func addPriority[T any, PT interface {
	*T
	model.HasMeta
}](items []T, priority int) []T {
	for i := range items { PT(&items[i]).GetMeta().Score += priority }
	return items
}

func scoreAndFilter[T any, PT interface {
	*T
	model.HasMeta
}](items []T, profileTags []string) []T {
	for i := range items {
		pMeta := PT(&items[i]).GetMeta()
		*pMeta = scoreMatchingTag(*pMeta, profileTags)
		if pMeta.Score == 0 && len(pMeta.Tags) > 0 {pMeta.Reason = model.Filtered}
	}
	return items
}

func scoreMatchingTag(meta model.Meta, profileTags []string) model.Meta {
	for tagValue, filterTag := range profileTags {
		for _, itemTag := range meta.Tags {
			if itemTag == filterTag {
				meta.Score += len(profileTags) - tagValue
				break
			}
		}
	}
	return meta
}