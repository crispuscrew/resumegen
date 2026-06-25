package usecase

import (
	"fmt"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// TrimIsNeeded reports whether the rendered page count exceeds the configured
// limit. A non-positive maxPages disables the gate.
func TrimIsNeeded(pageCount, maxPages float64) (bool, error) {
	if maxPages < 0 {
		return false, fmt.Errorf("invalid page limit %f: must be non-negative", maxPages)
	}
	if maxPages == 0 {
		return false, nil
	}
	return pageCount > maxPages, nil
}

// TrimLowest marks the lowest-scored included nested element (bullet or skill
// item) as Trimmed, then propagates that decision: any parent whose remaining
// included children fall below MinElements is also marked Trimmed. The bool
// return reports whether a trim took place.
func TrimLowest(data domain.ResumeData, minElements domain.MinElements) (domain.ResumeData, bool) {
	min := minIncluded(domain.FlatNested(data))
	if min == nil {
		return data, false
	}
	min.Reason = domain.Trimmed
	data = trimEmpty(data, minElements)
	return data, true
}

func minIncluded(metas []*domain.Meta) *domain.Meta {
	var min *domain.Meta
	for _, meta := range metas {
		if meta.Reason != domain.Included {
			continue
		}
		if min == nil || meta.Score < min.Score {
			min = meta
		}
	}
	return min
}

func trimEmpty(data domain.ResumeData, minElements domain.MinElements) domain.ResumeData {
	for i, job := range data.Jobs {
		if job.Reason != domain.Included {
			continue
		}
		n := 0
		for _, b := range job.Bullets {
			if b.Reason == domain.Included {
				n++
			}
		}
		if n < minElements.JobBullets {
			data.Jobs[i].Reason = domain.Trimmed
		}
	}

	for i, project := range data.Projects {
		if project.Reason != domain.Included {
			continue
		}
		n := 0
		for _, b := range project.Bullets {
			if b.Reason == domain.Included {
				n++
			}
		}
		if n < minElements.ProjectBullets {
			data.Projects[i].Reason = domain.Trimmed
		}
	}

	for i, cat := range data.SkillCats {
		if cat.Reason != domain.Included {
			continue
		}
		n := 0
		for _, item := range cat.Items {
			if item.Reason == domain.Included {
				n++
			}
		}
		if n < minElements.SkillItems {
			data.SkillCats[i].Reason = domain.Trimmed
		}
	}
	return data
}
