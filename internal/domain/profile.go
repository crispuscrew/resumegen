package domain

// Profile selects which content to include and in what priority order.
// Tags are listed highest-priority first.
type Profile struct {
	Tags   []string `toml:"tags"`
	Lang   string   `toml:"lang"`
	Output string   `toml:"output"`
}
