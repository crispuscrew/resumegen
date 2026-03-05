package types

type Profile struct {
	Tags   []string    `toml:"tags"`
	Lang   string      `toml:"lang"`
	Output string      `toml:"output"`
	Trim   []TrimGroup `toml:"trim"`
}

type TrimGroup struct {
	Tags []string `toml:"tags"`
}
