package config

type Config struct {
	Paths      Paths      `toml:"paths"`
	Render     Render     `toml:"render"`
	Estimation Estimation `toml:"estimation"`
}

type Paths struct {
	DataDir      string `toml:"data_dir"`
	ProfilesDir  string `toml:"profiles_dir"`
	TemplatesDir string `toml:"templates_dir"`
	OutputDir    string `toml:"output_dir"`
	TypstBin     string `toml:"typst_bin"`
}

type Render struct {
	PageLimit int `toml:"page_limit"`
}

type Estimation struct {
	PageHeight   float64 `toml:"page_height"`
	MarginTop    float64 `toml:"margin_top"`
	MarginBottom float64 `toml:"margin_bottom"`
	Safety       float64 `toml:"safety"`
	Section      float64 `toml:"section"`
	EntryHeader  float64 `toml:"entry_header"`
	Bullet       float64 `toml:"bullet"`
	SummaryLine  float64 `toml:"summary_line"`
	SkillLine    float64 `toml:"skill_line"`
}
