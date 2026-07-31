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
	return flatNested(data, false)
}

// FlatNestedVisible is FlatNested restricted to children of parents that are
// themselves Included. Scoring only filters an element on its own tags, so the
// bullets of a filtered-out job stay Included with score 0 and would otherwise
// be picked first by the trimmer - costing a full typst compile+query round
// trip each, without ever changing the page count.
func FlatNestedVisible(data ResumeData) []*Meta {
	return flatNested(data, true)
}

func flatNested(data ResumeData, visibleOnly bool) []*Meta {
	var metas []*Meta
	for i := range data.Jobs {
		if visibleOnly && data.Jobs[i].Reason != Included {
			continue
		}
		for j := range data.Jobs[i].Bullets {
			metas = append(metas, data.Jobs[i].Bullets[j].GetMeta())
		}
	}
	for i := range data.Projects {
		if visibleOnly && data.Projects[i].Reason != Included {
			continue
		}
		for j := range data.Projects[i].Bullets {
			metas = append(metas, data.Projects[i].Bullets[j].GetMeta())
		}
	}
	for i := range data.SkillCats {
		if visibleOnly && data.SkillCats[i].Reason != Included {
			continue
		}
		for j := range data.SkillCats[i].Items {
			metas = append(metas, data.SkillCats[i].Items[j].GetMeta())
		}
	}
	return metas
}
