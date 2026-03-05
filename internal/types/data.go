package types

// I18n holds a bilingual string pair used for all localizable fields.
type I18n struct {
	En string `toml:"en"`
	Ru string `toml:"ru"`
}

// ResumeData is the full in-memory representation of all resume content.
type ResumeData struct {
	Header    Header
	Jobs      []Job
	Projects  []Project
	Education []School
	Skills    []SkillCategory
}

// header.toml

type Header struct {
	Name     string    `toml:"name"`
	Contacts []Contact `toml:"contacts"`
	Summary  I18n      `toml:"summary"`
}

// Contact.Lang is optional: if set, the contact is shown only for that language.
type Contact struct {
	Lang  string `toml:"lang"`
	Value string `toml:"value"`
	Href  string `toml:"href"`
}

// experience.toml

type Job struct {
	Tags     []string `toml:"tags"`
	Title    I18n     `toml:"title"`
	Date     I18n     `toml:"date"`
	Company  string   `toml:"company"`
	Location I18n     `toml:"location"`
	Bullets  []Bullet `toml:"bullets"`
}

type Bullet struct {
	Tags []string `toml:"tags"`
	En   string   `toml:"en"`
	Ru   string   `toml:"ru"`
}

// projects.toml

type Project struct {
	Tags     []string `toml:"tags"`
	Title    string   `toml:"title"`
	Date     string   `toml:"date"`
	Subtitle string   `toml:"subtitle"`
	Detail   string   `toml:"detail"`
	Bullets  []Bullet `toml:"bullets"`
}

// education.toml

type School struct {
	Title    I18n `toml:"title"`
	Location I18n `toml:"location"`
	Degree   I18n `toml:"degree"`
	Date     I18n `toml:"date"`
}

// skills.toml

type SkillCategory struct {
	Name  I18n        `toml:"name"`
	Items []SkillItem `toml:"items"`
}

type SkillItem struct {
	Name string   `toml:"name"`
	Tags []string `toml:"tags"`
}
