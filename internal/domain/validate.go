package domain

import (
	"fmt"
	"sort"
	"unicode"
	"unicode/utf8"
)

// ValidateInput enforces input-safety rules over loaded resume data
// (DESIGN §4.2 step 1).
//
// NUL bytes are rejected regardless of strict — they have no legitimate use in
// resume data. When strict is true, the additional checks apply: control
// characters other than \n and \t are rejected, every string must be valid
// UTF-8, and the per-field-class byte limits are enforced. Zero fields in
// limits fall back to their defaults.
//
// Errors name the offending field path, e.g.
// "jobs[2].bullets[0].text[en]: 5012 bytes exceeds limit 4096".
func ValidateInput(data ResumeData, strict bool, limits Limits) error {
	v := inputValidator{strict: strict, limits: limits.withDefaults()}
	return v.resume(data)
}

type inputValidator struct {
	strict bool
	limits Limits
}

// check applies the rules to a single string at the given field path. limit is
// the byte ceiling for the field's class (ignored unless strict).
func (v inputValidator) check(field, value string, limit int) error {
	for i := 0; i < len(value); i++ {
		if value[i] == 0x00 {
			return fmt.Errorf("%s: NUL byte at offset %d", field, i)
		}
	}
	if !v.strict {
		return nil
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s: invalid UTF-8", field)
	}
	for _, r := range value {
		if r == '\n' || r == '\t' {
			continue
		}
		if unicode.IsControl(r) {
			return fmt.Errorf("%s: disallowed control character %#U", field, r)
		}
	}
	if len(value) > limit {
		return fmt.Errorf("%s: %d bytes exceeds limit %d", field, len(value), limit)
	}
	return nil
}

// i18n checks every language value of m. Languages are visited in sorted order
// so error messages are deterministic.
func (v inputValidator) i18n(field string, m I18n, limit int) error {
	langs := make([]string, 0, len(m))
	for lang := range m {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	for _, lang := range langs {
		if err := v.check(fmt.Sprintf("%s[%s]", field, lang), m[lang], limit); err != nil {
			return err
		}
	}
	return nil
}

// tags checks every tag of a Meta as a short identifier.
func (v inputValidator) tags(field string, m Meta) error {
	for i, t := range m.Tags {
		if err := v.check(fmt.Sprintf("%s.tags[%d]", field, i), t, v.limits.Short); err != nil {
			return err
		}
	}
	return nil
}

func (v inputValidator) resume(d ResumeData) error {
	if err := v.i18n("header.name", d.Header.Name, v.limits.Short); err != nil {
		return err
	}
	if err := v.i18n("header.summary", d.Header.Summary, v.limits.BulletText); err != nil {
		return err
	}
	for i, c := range d.Header.Contacts {
		base := fmt.Sprintf("header.contacts[%d]", i)
		if err := v.check(base+".lang", c.Lang, v.limits.Short); err != nil {
			return err
		}
		if err := v.check(base+".value", c.Value, v.limits.Short); err != nil {
			return err
		}
		if err := v.check(base+".href", c.Href, v.limits.URLOrPath); err != nil {
			return err
		}
	}

	for i, j := range d.Jobs {
		base := fmt.Sprintf("jobs[%d]", i)
		if err := v.tags(base, j.Meta); err != nil {
			return err
		}
		if err := v.i18n(base+".title", j.Title, v.limits.Short); err != nil {
			return err
		}
		if err := v.i18n(base+".date", j.Date, v.limits.Short); err != nil {
			return err
		}
		if err := v.i18n(base+".company", j.Company, v.limits.Short); err != nil {
			return err
		}
		if err := v.i18n(base+".location", j.Location, v.limits.Short); err != nil {
			return err
		}
		for k, b := range j.Bullets {
			bb := fmt.Sprintf("%s.bullets[%d]", base, k)
			if err := v.tags(bb, b.Meta); err != nil {
				return err
			}
			if err := v.i18n(bb+".text", b.Text, v.limits.BulletText); err != nil {
				return err
			}
		}
	}

	for i, p := range d.Projects {
		base := fmt.Sprintf("projects[%d]", i)
		if err := v.tags(base, p.Meta); err != nil {
			return err
		}
		if err := v.i18n(base+".title", p.Title, v.limits.Short); err != nil {
			return err
		}
		if err := v.i18n(base+".date", p.Date, v.limits.Short); err != nil {
			return err
		}
		if err := v.i18n(base+".subtitle", p.Subtitle, v.limits.Short); err != nil {
			return err
		}
		if err := v.i18n(base+".detail", p.Detail, v.limits.Short); err != nil {
			return err
		}
		for k, b := range p.Bullets {
			bb := fmt.Sprintf("%s.bullets[%d]", base, k)
			if err := v.tags(bb, b.Meta); err != nil {
				return err
			}
			if err := v.i18n(bb+".text", b.Text, v.limits.BulletText); err != nil {
				return err
			}
		}
	}

	for i, e := range d.Edu {
		base := fmt.Sprintf("edu[%d]", i)
		if err := v.i18n(base+".title", e.Title, v.limits.Short); err != nil {
			return err
		}
		if err := v.i18n(base+".degree", e.Degree, v.limits.Short); err != nil {
			return err
		}
		if err := v.i18n(base+".location", e.Location, v.limits.Short); err != nil {
			return err
		}
		if err := v.i18n(base+".date", e.Date, v.limits.Short); err != nil {
			return err
		}
	}

	for i, cat := range d.SkillCats {
		base := fmt.Sprintf("skills[%d]", i)
		if err := v.tags(base, cat.Meta); err != nil {
			return err
		}
		if err := v.i18n(base+".name", cat.Name, v.limits.Short); err != nil {
			return err
		}
		for k, item := range cat.Items {
			ii := fmt.Sprintf("%s.items[%d]", base, k)
			if err := v.tags(ii, item.Meta); err != nil {
				return err
			}
			if err := v.i18n(ii+".name", item.Name, v.limits.Short); err != nil {
				return err
			}
		}
	}

	return nil
}
