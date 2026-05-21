package domain

// Config is the parsed config.toml. Adapter layer parses TOML into this;
// usecase layer consumes it as a pure value.
type Config struct {
	Paths  Paths  `toml:"paths"`
	Render Render `toml:"render"`
	Score  Score  `toml:"score"`
}

type Paths struct {
	OutputDir string `toml:"output_dir"`
	TypstBin  string `toml:"typst_bin"`
}

type Score struct {
	SkillPriority int `toml:"skill_priority"`
}

type Render struct {
	PageLimit    float64     `toml:"page_limit"`
	PageHeightPt float64     `toml:"page_height_pt"`
	MinElements  MinElements `toml:"min_elements"`

	// ForceUnsafe switches the sanitizer to permissive mode: malformed
	// markup or disallowed URL schemes no longer fail the render — the
	// offending bullet is emitted as Typst-escaped literal text instead.
	// CLI: --force. Default false (strict).
	ForceUnsafe bool `toml:"force_unsafe"`
}

type MinElements struct {
	JobBullets     int `toml:"job_bullets"`
	ProjectBullets int `toml:"project_bullets"`
	SkillItems     int `toml:"skill_items"`
}
