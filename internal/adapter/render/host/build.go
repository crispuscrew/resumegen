package host

import (
	"bytes"
	"fmt"

	"github.com/crispuscrew/resumegen/internal/adapter/render/sanitize"
	"github.com/crispuscrew/resumegen/internal/domain"
)

// BuildTypstSource serializes scored resume data into the Typst `#let` block
// consumed by templates/resume.typ. Every value emitted between `[...]`
// content brackets passes through the sanitizer first — those are the
// injection-relevant positions. Values emitted via `%q` are Go string
// literals and need no further escaping because their consumer is a Typst
// string literal, never markup.
//
// mode picks between strict failure and permissive (literal-text) fallback
// when the sanitizer rejects an input.
func BuildTypstSource(data domain.ResumeData, profile domain.Profile, mode sanitize.Mode) ([]byte, error) {
	var buf bytes.Buffer

	if err := emitHeader(&buf, data.Header, profile, mode); err != nil {
		return nil, err
	}
	if err := emitJobs(&buf, data.Jobs, profile, mode); err != nil {
		return nil, err
	}
	if err := emitProjects(&buf, data.Projects, profile, mode); err != nil {
		return nil, err
	}
	if err := emitSkills(&buf, data.SkillCats, profile, mode); err != nil {
		return nil, err
	}
	emitEdu(&buf, data.Edu, profile)
	return buf.Bytes(), nil
}

func sanitizeOrAnnotate(field, value string, mode sanitize.Mode) (string, error) {
	out, err := sanitize.Sanitize(value, mode)
	if err != nil {
		return "", fmt.Errorf("sanitize %s: %w", field, err)
	}
	return out, nil
}

func emitHeader(buf *bytes.Buffer, h domain.Header, profile domain.Profile, mode sanitize.Mode) error {
	fmt.Fprintf(buf, "#let r-lang = %q\n", profile.Lang)
	fmt.Fprintf(buf, "#let r-name = %q\n", h.Name.Lang(profile.Lang))

	summary, err := sanitizeOrAnnotate("header.summary", h.Summary.Lang(profile.Lang), mode)
	if err != nil {
		return err
	}
	fmt.Fprintf(buf, "#let r-summary = [%s]\n", summary)

	buf.WriteString("#let r-contacts = (\n")
	for _, c := range h.Contacts {
		if c.Lang == "" || c.Lang == profile.Lang {
			fmt.Fprintf(buf, "	(value : %q, href : %q), \n", c.Value, c.Href)
		}
	}
	buf.WriteString(")\n")
	return nil
}

func emitJobs(buf *bytes.Buffer, jobs []domain.Job, profile domain.Profile, mode sanitize.Mode) error {
	buf.WriteString("#let r-jobs = (\n")
	for _, job := range jobs {
		if job.Reason != domain.Included {
			continue
		}
		fmt.Fprintf(buf, "	(title : %q, date : %q, company : %q, location : %q, bullets : (",
			job.Title.Lang(profile.Lang),
			job.Date.Lang(profile.Lang),
			job.Company.Lang(profile.Lang),
			job.Location.Lang(profile.Lang))
		for _, b := range job.Bullets {
			if b.Reason != domain.Included {
				continue
			}
			text, err := sanitizeOrAnnotate(fmt.Sprintf("job %q bullet", job.Company.Lang(profile.Lang)), b.Text.Lang(profile.Lang), mode)
			if err != nil {
				return err
			}
			fmt.Fprintf(buf, "\n		[%s],", text)
		}
		buf.WriteString(")),\n")
	}
	buf.WriteString(")\n")
	return nil
}

func emitProjects(buf *bytes.Buffer, projects []domain.Project, profile domain.Profile, mode sanitize.Mode) error {
	buf.WriteString("#let r-projects = (\n")
	for _, p := range projects {
		if p.Reason != domain.Included {
			continue
		}
		fmt.Fprintf(buf, "	(title : %q, date : %q, subtitle : %q, detail : %q, bullets : (",
			p.Title.Lang(profile.Lang),
			p.Date.Lang(profile.Lang),
			p.Subtitle.Lang(profile.Lang),
			p.Detail.Lang(profile.Lang))
		for _, b := range p.Bullets {
			if b.Reason != domain.Included {
				continue
			}
			text, err := sanitizeOrAnnotate(fmt.Sprintf("project %q bullet", p.Title.Lang(profile.Lang)), b.Text.Lang(profile.Lang), mode)
			if err != nil {
				return err
			}
			fmt.Fprintf(buf, "\n		[%s],", text)
		}
		buf.WriteString(")),\n")
	}
	buf.WriteString(")\n")
	return nil
}

func emitSkills(buf *bytes.Buffer, cats []domain.SkillCat, profile domain.Profile, mode sanitize.Mode) error {
	buf.WriteString("#let r-skills = (\n")
	for _, cat := range cats {
		if cat.Reason != domain.Included {
			continue
		}
		fmt.Fprintf(buf, "	(category : %q, items : (", cat.Name.Lang(profile.Lang))
		for _, item := range cat.Items {
			if item.Reason != domain.Included {
				continue
			}
			text, err := sanitizeOrAnnotate(fmt.Sprintf("skill cat %q item", cat.Name.Lang(profile.Lang)), item.Name.Lang(profile.Lang), mode)
			if err != nil {
				return err
			}
			fmt.Fprintf(buf, "[%s],", text)
		}
		buf.WriteString(")),\n")
	}
	buf.WriteString(")\n")
	return nil
}

func emitEdu(buf *bytes.Buffer, edu []domain.Edu, profile domain.Profile) {
	buf.WriteString("#let r-edu = (\n")
	for _, e := range edu {
		fmt.Fprintf(buf, "	(title : %q, degree : %q, location : %q, date : %q),\n",
			e.Title.Lang(profile.Lang),
			e.Degree.Lang(profile.Lang),
			e.Location.Lang(profile.Lang),
			e.Date.Lang(profile.Lang))
	}
	buf.WriteString(")\n")
}
