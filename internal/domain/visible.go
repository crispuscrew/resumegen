package domain

// VisibleResume returns a copy of data containing only the entities that the
// renderer actually emits: every job, project, skill category, bullet, and
// skill item whose Reason is Included. Education is always shown in full, and
// the header is never filtered. Ordering is preserved.
//
// This is the single source of truth for "what went into the PDF" - both the
// Markdown emitter and the filtered-TOML emitter project from it, so they can
// never drift from the render path's Reason == Included rule.
func VisibleResume(data ResumeData) ResumeData {
	out := ResumeData{
		Header: data.Header,
		Edu:    data.Edu,
	}

	for _, job := range data.Jobs {
		if job.Reason != Included {
			continue
		}
		job.Bullets = visibleBullets(job.Bullets)
		out.Jobs = append(out.Jobs, job)
	}

	for _, p := range data.Projects {
		if p.Reason != Included {
			continue
		}
		p.Bullets = visibleBullets(p.Bullets)
		out.Projects = append(out.Projects, p)
	}

	for _, cat := range data.SkillCats {
		if cat.Reason != Included {
			continue
		}
		var items []SkillItem
		for _, item := range cat.Items {
			if item.Reason == Included {
				items = append(items, item)
			}
		}
		cat.Items = items
		out.SkillCats = append(out.SkillCats, cat)
	}

	return out
}

func visibleBullets(bullets []Bullet) []Bullet {
	var out []Bullet
	for _, b := range bullets {
		if b.Reason == Included {
			out = append(out, b)
		}
	}
	return out
}
