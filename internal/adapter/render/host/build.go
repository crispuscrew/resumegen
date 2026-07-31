package host

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/crispuscrew/resumegen/internal/adapter/render/sanitize"
	"github.com/crispuscrew/resumegen/internal/domain"
)

// BuildTypstSource serializes scored resume data into the Typst `#let` block
// consumed by templates/resume.typ. Every value emitted between `[...]`
// content brackets passes through the sanitizer first - those are the
// injection-relevant positions. Values in string positions go through typstStr
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

// typstStr renders s as a valid Typst string literal. Go %q is ALMOST right
// (it escapes backslash and quote identically) but writes control characters
// as \x07 / \u0085, which Typst does not understand - with strict_input off
// such a rune would produce an invalid escape and a confusing compile error.
// Typst wants \u{...}, so emit that; printable runes pass through raw.
func typstStr(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString("\\\\")
		case r == '"':
			b.WriteString("\\\"")
		case r == '\n':
			b.WriteString("\\n")
		case r == '\t':
			b.WriteString("\\t")
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, "\\u{%x}", r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
func emitHeader(buf *bytes.Buffer, h domain.Header, profile domain.Profile, mode sanitize.Mode) error {
	fmt.Fprintf(buf, "#let r-lang = %s\n", typstStr(profile.Lang))
	fmt.Fprintf(buf, "#let r-name = %s\n", typstStr(h.Name.Lang(profile.Lang)))

	summary, err := sanitizeOrAnnotate("header.summary", h.Summary.Lang(profile.Lang), mode)
	if err != nil {
		return err
	}
	fmt.Fprintf(buf, "#let r-summary = [%s]\n", summary)

	buf.WriteString("#let r-contacts = (\n")
	for _, c := range h.Contacts {
		if c.Lang == "" || c.Lang == profile.Lang {
			// The href lands in the template's link(...) and becomes a /URI in
			// the PDF, so it needs the same scheme allowlist the sanitizer
			// applies to links inside bullet text; otherwise javascript: and
			// file:// hrefs sail straight through. An empty href is the
			// documented "plain text, not a link" case.
			if c.Href != "" {
				if err := sanitize.ValidateURL(c.Href); err != nil {
					return fmt.Errorf("header contact %q: %w", c.Value, err)
				}
			}
			fmt.Fprintf(buf, "	(value : %s, href : %s), \n", typstStr(c.Value), typstStr(c.Href))
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
		fmt.Fprintf(buf, "	(title : %s, date : %s, company : %s, location : %s, bullets : (",
			typstStr(job.Title.Lang(profile.Lang)),
			typstStr(job.Date.Lang(profile.Lang)),
			typstStr(job.Company.Lang(profile.Lang)),
			typstStr(job.Location.Lang(profile.Lang)))
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
		fmt.Fprintf(buf, "	(title : %s, date : %s, subtitle : %s, detail : %s, bullets : (",
			typstStr(p.Title.Lang(profile.Lang)),
			typstStr(p.Date.Lang(profile.Lang)),
			typstStr(p.Subtitle.Lang(profile.Lang)),
			typstStr(p.Detail.Lang(profile.Lang)))
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
		fmt.Fprintf(buf, "	(category : %s, items : (", typstStr(cat.Name.Lang(profile.Lang)))
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
		fmt.Fprintf(buf, "	(title : %s, degree : %s, location : %s, date : %s),\n",
			typstStr(e.Title.Lang(profile.Lang)),
			typstStr(e.Degree.Lang(profile.Lang)),
			typstStr(e.Location.Lang(profile.Lang)),
			typstStr(e.Date.Lang(profile.Lang)))
	}
	buf.WriteString(")\n")
}
