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

	// StripMetadata enables an opt-in qpdf post-process that empties the
	// rendered PDF's /Author, /Creator, /Producer, /CreationDate, /ModDate.
	// Default false — existing users without qpdf installed are unaffected.
	StripMetadata bool `toml:"strip_metadata"`

	// StrictInput enables opt-in input validation at load time (§4.2 step 1).
	// NUL bytes are rejected regardless; when true, control characters (except
	// \n and \t), invalid UTF-8, and the per-field-class byte limits below are
	// enforced too. Default false, so existing v1.0 data loads unchanged.
	StrictInput bool `toml:"strict_input"`

	// Limits holds per-field-class byte limits, enforced only when StrictInput
	// is true. Any zero field falls back to its default (see Limits.withDefaults).
	Limits Limits `toml:"limits"`
}

// Limits are per-field-class byte limits enforced when Render.StrictInput is
// true. A zero field means "use the default" (see withDefaults).
type Limits struct {
	Short      int `toml:"short"`       // names, titles, dates, company, location, tags
	BulletText int `toml:"bullet_text"` // bullet text and the header summary
	Notes      int `toml:"notes"`       // application notes (v1.5); reserved
	URLOrPath  int `toml:"url_or_path"` // contact hrefs and path-like fields
}

// Default per-field-class byte limits (DESIGN §4.2 step 1).
const (
	defaultLimitShort      = 256
	defaultLimitBulletText = 4096
	defaultLimitNotes      = 65536
	defaultLimitURLOrPath  = 2048
)

// withDefaults returns a copy of l with each zero field replaced by its default.
func (l Limits) withDefaults() Limits {
	if l.Short == 0 {
		l.Short = defaultLimitShort
	}
	if l.BulletText == 0 {
		l.BulletText = defaultLimitBulletText
	}
	if l.Notes == 0 {
		l.Notes = defaultLimitNotes
	}
	if l.URLOrPath == 0 {
		l.URLOrPath = defaultLimitURLOrPath
	}
	return l
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
