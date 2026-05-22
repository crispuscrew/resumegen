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

	// UseContainer selects the renderer backend. Valid values:
	//   ""      — host typst binary (v1.0 behavior; default; byte-equiv)
	//   "false" — same as ""
	//   "true"  — require a container engine; fail loud if none found
	//   "auto"  — use container if engine present, else host
	// The container backend uses rootless podman if available, falling back
	// to docker. See ContainerMode for the parsed form.
	UseContainer string `toml:"use_container"`
}

// ContainerMode is the parsed form of Render.UseContainer.
type ContainerMode int

const (
	ContainerOff ContainerMode = iota
	ContainerOn
	ContainerAuto
)

// ParseContainerMode converts the raw config string to ContainerMode.
// Empty string and "false" both mean off. Unknown values are reported.
func ParseContainerMode(s string) (ContainerMode, error) {
	switch s {
	case "", "false":
		return ContainerOff, nil
	case "true":
		return ContainerOn, nil
	case "auto":
		return ContainerAuto, nil
	default:
		return ContainerOff, &InvalidContainerModeError{Value: s}
	}
}

type InvalidContainerModeError struct{ Value string }

func (e *InvalidContainerModeError) Error() string {
	return "render.use_container must be one of \"\", \"true\", \"false\", \"auto\"; got " + quote(e.Value)
}

func quote(s string) string { return "\"" + s + "\"" }

type MinElements struct {
	JobBullets     int `toml:"job_bullets"`
	ProjectBullets int `toml:"project_bullets"`
	SkillItems     int `toml:"skill_items"`
}
