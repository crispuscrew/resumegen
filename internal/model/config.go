package model

type Config struct {
	Paths      		Paths  		`toml:"paths"`
	Render     		Render 		`toml:"render"`
	Score			Score		`toml:"score"`
}

type Paths struct {
	OutputDir		string 		`toml:"output_dir"`
	TypstBin		string 		`toml:"typst_bin"`
}

type Score struct {
	SkillPriority	int			`toml:"skill_priority"`
}

type Render struct {
	PageLimit 		float64		`toml:"page_limit"`
	PageHeightPt	float64		`toml:"page_height_pt"`
	MinElements		MinElements	`toml:"min_elements"`
}

type MinElements struct {
	JobBullets		int			`toml:"job_bullets"`
	ProjectBullets	int			`toml:"project_bullets"`
	SkillItems		int			`toml:"skill_items"`
}