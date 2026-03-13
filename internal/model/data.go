package model

type I18n map[string]string
func (i I18n) Lang(lang string) string {
	if v, ok := i[lang]; ok { return v }
	return i["en"] // fallback
}

// ResumeData is the full in-memory representation of all resume content.
type ResumeData struct {
	Header    	Header
	Jobs      	[]Job
	Projects  	[]Project
	Edu 		[]Edu
	SkillCats	[]SkillCat
}

type Reason int
const (
	Included  Reason = iota
	Filtered
	Trimmed
) 

type Meta struct {
	Tags		[]string	`toml:"tags"`
	Score		int
	Reason		Reason
}
type HasMeta interface { GetMeta() *Meta }
func (m *Meta) GetMeta() *Meta { return m }

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

// jobs.toml

type Job struct {
	Meta
	Bullets   	[]Bullet 	`toml:"bullets"`
	Title		I18n     	`toml:"title"`
	Date		I18n     	`toml:"date"`
	Company  	string   	`toml:"company"`
	Location 	I18n     	`toml:"location"`
}

type Bullet struct {
	Meta
	Text 		I18n 		`toml:"text"`
}

// projects.toml

type Project struct {
	Meta
	Bullets   	[]Bullet 	`toml:"bullets"`
	Title    	string   	`toml:"title"`
	Date     	string   	`toml:"date"`
	Subtitle 	string   	`toml:"subtitle"`
	Detail   	string   	`toml:"detail"`
}

// education.toml

type Edu struct {
	Title    	I18n 		`toml:"title"`
	Location 	I18n 		`toml:"location"`
	Degree   	I18n 		`toml:"degree"`
	Date     	I18n 		`toml:"date"`
}

// skills.toml

type SkillCat struct {
	Meta
	Name  		I18n        `toml:"name"`
	Items 		[]SkillItem `toml:"items"`
}

type SkillItem struct {
	Meta
	Name 		string   	`toml:"name"`
}

func FlatTopLevel(data ResumeData) []*Meta {                                                                                              
	var metas []*Meta                                                                                                                           
	for i := range data.Jobs     { metas = append(metas, data.Jobs[i].GetMeta())     }                                                                
	for i := range data.Projects { metas = append(metas, data.Projects[i].GetMeta()) }                                                                
	for i := range data.SkillCats{ metas = append(metas, data.SkillCats[i].GetMeta())}
	return metas                                                                                                                                      
}

func FlatNested(data ResumeData) []*Meta {
	var metas []*Meta
	for i := range data.Jobs {
		for j := range data.Jobs[i].Bullets { metas = append(metas, data.Jobs[i].Bullets[j].GetMeta()) }
	}
	for i := range data.Projects {
		for j := range data.Projects[i].Bullets { metas = append(metas, data.Projects[i].Bullets[j].GetMeta()) }
	}
	for i := range data.SkillCats {
		for j := range data.SkillCats[i].Items { metas = append(metas, data.SkillCats[i].Items[j].GetMeta()) }
	}
	return metas
}