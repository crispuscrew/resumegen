package model

type Config struct {
	Paths      		Paths  		`toml:"paths"`
	Render     		Render 		`toml:"render"`
}

type Paths struct {
	OutputDir		string 		`toml:"output_dir"`
	TypstBin		string 		`toml:"typst_bin"`
}

type Render struct {
	PageLimit 		float64		`toml:"page_limit"`
	MinElements		MinElements	`toml:"min_elements"`
}

type MinElements struct {
	JobBullets		int			`toml:"job_bullets"`
	ProjectBullets	int			`toml:"project_bullets"`
	SkillCats		int			`toml:"skill_cats"`
}