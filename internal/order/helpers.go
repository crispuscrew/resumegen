package order

import (
	"github.com/crispuscrew/resumegen/internal/model"

	"cmp"
	"slices"
)

func scoreMatchingTag(tagging model.Tagging, profileTags []string) model.Tagging {
	for tagValue, filterTag := range profileTags {
		for _, itemTag := range tagging.Tags {
			if itemTag == filterTag {
				tagging.Score += len(profileTags) - tagValue
				break
			}
		}
	}
	return tagging
}

// I hate it, but it better that rewrite data structure to be more generic and reusable. Maybe in the future

func scoreAndSortBullets(bullets []model.Bullet, profileTags []string) []model.Bullet {
	for i := range bullets {
		bullets[i].Tagging = scoreMatchingTag(bullets[i].Tagging, profileTags)
	}
	slices.SortFunc(bullets, func(a, b model.Bullet) int {
		return cmp.Compare(b.Score, a.Score)
	})
	i := slices.IndexFunc(bullets, func(b model.Bullet) bool { return b.Score == 0 })
	if i != -1 { bullets = bullets[:i] }
	return bullets
}

func scoreAndSortJobs(jobs []model.Job, profileTags []string) []model.Job {
	for i := range jobs {
		jobs[i].Tagging = scoreMatchingTag(jobs[i].Tagging, profileTags)
	}
	slices.SortFunc(jobs, func(a, b model.Job) int {
		return cmp.Compare(b.Score, a.Score)
	})
	i := slices.IndexFunc(jobs, func(j model.Job) bool { return j.Score == 0 })
	if i != -1 { jobs = jobs[:i] }
	return jobs
}

func scoreAndSortProjects(projects []model.Project, profileTags []string) []model.Project {
	for i := range projects {
		projects[i].Tagging = scoreMatchingTag(projects[i].Tagging, profileTags)
	}
	slices.SortFunc(projects, func(a, b model.Project) int {
		return cmp.Compare(b.Score, a.Score)
	})
	i := slices.IndexFunc(projects, func(p model.Project) bool { return p.Score == 0 })
	if i != -1 { projects = projects[:i] }
	return projects
}

func scoreAndSortSkills(skills []model.SkillItem, profileTags []string) []model.SkillItem {
	for i := range skills {
		skills[i].Tagging = scoreMatchingTag(skills[i].Tagging, profileTags)
	}
	slices.SortFunc(skills, func(a, b model.SkillItem) int {
		return cmp.Compare(b.Score, a.Score)
	})
	i := slices.IndexFunc(skills, func(s model.SkillItem) bool { return s.Score == 0 })
	if i != -1 { skills = skills[:i] }
	return skills
}