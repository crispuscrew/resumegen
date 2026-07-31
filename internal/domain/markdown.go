package domain

import (
	"strings"
)

// RenderMarkdown projects the visible resume into a Markdown document at the
// profile's language. It is meant to be pasted into an LLM alongside a job
// description, so it carries the human text - including the authored inline
// markup (*bold*, #link(...)) - rather than the sanitizer's render output.
//
// Only Included entities appear and ordering matches the PDF (see
// VisibleResume). Output is deterministic.
func RenderMarkdown(data ResumeData, profile Profile) []byte {
	v := VisibleResume(data)
	lang := profile.Lang

	var b strings.Builder

	if name := v.Header.Name.Lang(lang); name != "" {
		b.WriteString("# " + name + "\n\n")
	}
	for _, c := range v.Header.Contacts {
		if c.Lang == "" || c.Lang == lang {
			b.WriteString("- " + c.Value)
			if c.Href != "" {
				b.WriteString(" (" + c.Href + ")")
			}
			b.WriteString("\n")
		}
	}
	if summary := v.Header.Summary.Lang(lang); summary != "" {
		b.WriteString("\n" + summary + "\n")
	}

	if len(v.Jobs) > 0 {
		b.WriteString("\n## Experience\n")
		for _, job := range v.Jobs {
			b.WriteString("\n### " + heading(job.Title.Lang(lang), job.Company.Lang(lang)) + "\n")
			writeMeta(&b, job.Date.Lang(lang), job.Location.Lang(lang))
			writeBullets(&b, job.Bullets, lang)
		}
	}

	if len(v.Projects) > 0 {
		b.WriteString("\n## Projects\n")
		for _, p := range v.Projects {
			b.WriteString("\n### " + heading(p.Title.Lang(lang), p.Subtitle.Lang(lang)) + "\n")
			writeMeta(&b, p.Date.Lang(lang), p.Detail.Lang(lang))
			writeBullets(&b, p.Bullets, lang)
		}
	}

	if len(v.SkillCats) > 0 {
		b.WriteString("\n## Skills\n\n")
		for _, cat := range v.SkillCats {
			var names []string
			for _, item := range cat.Items {
				names = append(names, item.Name.Lang(lang))
			}
			b.WriteString("- " + cat.Name.Lang(lang) + ": " + strings.Join(names, ", ") + "\n")
		}
	}

	if len(v.Edu) > 0 {
		b.WriteString("\n## Education\n")
		for _, e := range v.Edu {
			b.WriteString("\n### " + e.Title.Lang(lang) + "\n")
			writeMeta(&b, e.Degree.Lang(lang), e.Location.Lang(lang), e.Date.Lang(lang))
		}
	}

	return []byte(b.String())
}

// heading joins a title with an optional secondary part as "title - second".
// Either part may be missing; the other stands alone.
func heading(title, second string) string {
	switch {
	case title == "":
		return second
	case second == "":
		return title
	}
	return title + " - " + second
}

// writeMeta emits a non-empty " | "-joined metadata line, if any part is set.
func writeMeta(b *strings.Builder, parts ...string) {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	if len(nonEmpty) > 0 {
		b.WriteString(strings.Join(nonEmpty, " | ") + "\n")
	}
}

func writeBullets(b *strings.Builder, bullets []Bullet, lang string) {
	for _, bl := range bullets {
		b.WriteString("- " + bl.Text.Lang(lang) + "\n")
	}
}
