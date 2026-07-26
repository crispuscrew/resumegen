package domain

// skills.toml

type SkillCat struct {
	Meta
	Name  I18n        `toml:"name"`
	Items []SkillItem `toml:"items"`
}

type SkillItem struct {
	Meta
	Name I18n `toml:"name"`
}
