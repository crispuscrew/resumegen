package usecase

import "github.com/crispuscrew/resumegen/internal/domain"

// Score applies tag-driven scoring to every scoreable element in data and
// marks elements with no matching tag as Filtered. Skill items receive an
// extra priority bump per cfgScore.SkillPriority.
func Score(data domain.ResumeData, profileTags []string, cfgScore domain.Score) domain.ResumeData {
	data.Jobs = scoreAndFilter[domain.Job, *domain.Job](data.Jobs, profileTags)
	for i := range data.Jobs {
		data.Jobs[i].Bullets = scoreAndFilter[domain.Bullet, *domain.Bullet](data.Jobs[i].Bullets, profileTags)
	}

	data.Projects = scoreAndFilter[domain.Project, *domain.Project](data.Projects, profileTags)
	for i := range data.Projects {
		data.Projects[i].Bullets = scoreAndFilter[domain.Bullet, *domain.Bullet](data.Projects[i].Bullets, profileTags)
	}

	data.SkillCats = scoreAndFilter[domain.SkillCat, *domain.SkillCat](data.SkillCats, profileTags)
	for i := range data.SkillCats {
		data.SkillCats[i].Items = scoreAndFilter[domain.SkillItem, *domain.SkillItem](data.SkillCats[i].Items, profileTags)
		data.SkillCats[i].Items = addPriority[domain.SkillItem, *domain.SkillItem](data.SkillCats[i].Items, cfgScore.SkillPriority)
	}
	return data
}

func addPriority[T any, PT interface {
	*T
	domain.HasMeta
}](items []T, priority int) []T {
	for i := range items {
		PT(&items[i]).GetMeta().Score += priority
	}
	return items
}

func scoreAndFilter[T any, PT interface {
	*T
	domain.HasMeta
}](items []T, profileTags []string) []T {
	for i := range items {
		pMeta := PT(&items[i]).GetMeta()
		*pMeta = scoreMatchingTag(*pMeta, profileTags)
		if pMeta.Score == 0 && len(pMeta.Tags) > 0 {
			pMeta.Reason = domain.Filtered
		}
	}
	return items
}

func scoreMatchingTag(meta domain.Meta, profileTags []string) domain.Meta {
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
