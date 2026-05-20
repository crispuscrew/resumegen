package domain

// Reason explains why a scored element is included, filtered, or trimmed.
type Reason int

const (
	Included Reason = iota
	Filtered
	Trimmed
)

// Meta is the scoring metadata embedded into Job/Project/Bullet/SkillCat/SkillItem.
type Meta struct {
	Tags   []string `toml:"tags"`
	Score  int
	Reason Reason
}

// HasMeta lets pointer-receiver helpers act uniformly on tagged entities.
type HasMeta interface {
	GetMeta() *Meta
}

// GetMeta satisfies HasMeta for *Meta itself and, by promotion, for any
// struct that embeds Meta.
func (m *Meta) GetMeta() *Meta { return m }
