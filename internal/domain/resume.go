package domain

// ResumeData is the full in-memory representation of all resume content.
type ResumeData struct {
	Header    Header
	Jobs      []Job
	Projects  []Project
	Edu       []Edu
	SkillCats []SkillCat
}

// header.toml
type Header struct {
	Name     I18n      `toml:"name"`
	Contacts []Contact `toml:"contacts"`
	Summary  I18n      `toml:"summary"`
}

// Contact.Lang is optional: if set, the contact is shown only for that language.
type Contact struct {
	Lang  string `toml:"lang"`
	Value string `toml:"value"`
	Href  string `toml:"href"`
}

// jobs.toml
type Job struct {
	Meta
	Bullets  []Bullet `toml:"bullets"`
	Title    I18n     `toml:"title"`
	Date     I18n     `toml:"date"`
	Company  I18n     `toml:"company"`
	Location I18n     `toml:"location"`
}

type Bullet struct {
	Meta
	Text I18n `toml:"text"`
}

// projects.toml
type Project struct {
	Meta
	Bullets  []Bullet `toml:"bullets"`
	Title    I18n     `toml:"title"`
	Date     I18n     `toml:"date"`
	Subtitle I18n     `toml:"subtitle"`
	Detail   I18n     `toml:"detail"`
}

// education.toml
type Edu struct {
	Title    I18n `toml:"title"`
	Location I18n `toml:"location"`
	Degree   I18n `toml:"degree"`
	Date     I18n `toml:"date"`
}

// FlatTopLevel returns the Meta pointers of every top-level scoreable entity
// (jobs, projects, skill categories) in iteration order.
func FlatTopLevel(data ResumeData) []*Meta {
	var metas []*Meta
	for i := range data.Jobs {
		metas = append(metas, data.Jobs[i].GetMeta())
	}
	for i := range data.Projects {
		metas = append(metas, data.Projects[i].GetMeta())
	}
	for i := range data.SkillCats {
		metas = append(metas, data.SkillCats[i].GetMeta())
	}
	return metas
}

// FlatNested returns the Meta pointers of every nested scoreable entity
// (bullets of jobs and projects, items of skill categories).
func FlatNested(data ResumeData) []*Meta {
	var metas []*Meta
	for i := range data.Jobs {
		for j := range data.Jobs[i].Bullets {
			metas = append(metas, data.Jobs[i].Bullets[j].GetMeta())
		}
	}
	for i := range data.Projects {
		for j := range data.Projects[i].Bullets {
			metas = append(metas, data.Projects[i].Bullets[j].GetMeta())
		}
	}
	for i := range data.SkillCats {
		for j := range data.SkillCats[i].Items {
			metas = append(metas, data.SkillCats[i].Items[j].GetMeta())
		}
	}
	return metas
}
