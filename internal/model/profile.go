package model

type Profile struct {
	Tags   []string `toml:"tags"` // highest priority first, lowest last
	Lang   string   `toml:"lang"`
	Output string   `toml:"output"`
}
