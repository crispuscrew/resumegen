package host

import (
	"bytes"
	"fmt"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// BuildTypstSource serializes scored resume data into the Typst `#let` block
// consumed by templates/resume.typ. Output is deterministic and byte-stable
// across runs given identical input — used as a golden-file gate.
func BuildTypstSource(data domain.ResumeData, profile domain.Profile) ([]byte, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "#let r-lang = %q\n", profile.Lang)
	fmt.Fprintf(&buf, "#let r-name = %q\n", data.Header.Name.Lang(profile.Lang))
	fmt.Fprintf(&buf, "#let r-summary = [%s]\n", data.Header.Summary.Lang(profile.Lang))

	// contacts
	buf.WriteString("#let r-contacts = (\n")
	for _, contact := range data.Header.Contacts {
		if contact.Lang == "" || contact.Lang == profile.Lang {
			fmt.Fprintf(&buf, "	(value : %q, href : %q), \n", contact.Value, contact.Href)
		}
	}
	buf.WriteString(")\n")

	// jobs
	buf.WriteString("#let r-jobs = (\n")
	for _, job := range data.Jobs {
		if job.Reason != domain.Included {
			continue
		}
		fmt.Fprintf(&buf, "	(title : %q, date : %q, company : %q, location : %q, bullets : (",
			job.Title.Lang(profile.Lang),
			job.Date.Lang(profile.Lang),
			job.Company.Lang(profile.Lang),
			job.Location.Lang(profile.Lang))
		for _, bullet := range job.Bullets {
			if bullet.Reason != domain.Included {
				continue
			}
			fmt.Fprintf(&buf, "\n		[%s],", bullet.Text.Lang(profile.Lang))
		}
		buf.WriteString(")),\n")
	}
	buf.WriteString(")\n")

	// projects
	buf.WriteString("#let r-projects = (\n")
	for _, project := range data.Projects {
		if project.Reason != domain.Included {
			continue
		}
		fmt.Fprintf(&buf, "	(title : %q, date : %q, subtitle : %q, detail : %q, bullets : (",
			project.Title.Lang(profile.Lang),
			project.Date.Lang(profile.Lang),
			project.Subtitle.Lang(profile.Lang),
			project.Detail.Lang(profile.Lang))
		for _, bullet := range project.Bullets {
			if bullet.Reason != domain.Included {
				continue
			}
			fmt.Fprintf(&buf, "\n		[%s],", bullet.Text.Lang(profile.Lang))
		}
		buf.WriteString(")),\n")
	}
	buf.WriteString(")\n")

	// skills
	buf.WriteString("#let r-skills = (\n")
	for _, cat := range data.SkillCats {
		if cat.Reason != domain.Included {
			continue
		}
		fmt.Fprintf(&buf, "	(category : %q, items : (", cat.Name.Lang(profile.Lang))
		for _, item := range cat.Items {
			if item.Reason != domain.Included {
				continue
			}
			fmt.Fprintf(&buf, "[%s],", item.Name.Lang(profile.Lang))
		}
		buf.WriteString(")),\n")
	}
	buf.WriteString(")\n")

	// edu
	buf.WriteString("#let r-edu = (\n")
	for _, edu := range data.Edu {
		fmt.Fprintf(&buf, "	(title : %q, degree : %q, location : %q, date : %q),\n",
			edu.Title.Lang(profile.Lang),
			edu.Degree.Lang(profile.Lang),
			edu.Location.Lang(profile.Lang),
			edu.Date.Lang(profile.Lang))
	}
	buf.WriteString(")\n")

	return buf.Bytes(), nil
}
