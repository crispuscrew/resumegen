package domain

// Config is the parsed config.toml. Adapter layer parses TOML into this;
// usecase layer consumes it as a pure value.
type Config struct {
	Paths   Paths   `toml:"paths"`
	Render  Render  `toml:"render"`
	Score   Score   `toml:"score"`
	Tracker Tracker `toml:"tracker"`
	TUI     TUI     `toml:"tui"`
}

// TUI holds interactive terminal-UI settings (v1.5). The section is optional; an
// absent [tui] leaves the default theme. Only "default" ships today - the key
// exists so user-contributed palettes can be added without a config change.
type TUI struct {
	// Theme selects the color palette. "" or an unknown value resolve to the
	// built-in "default" theme (see ResolvedTheme).
	Theme string `toml:"theme"`
}

// knownThemes are the shipped TUI themes. Only "default" exists in v1.5.
var knownThemes = map[string]struct{}{"default": {}}

// ResolvedTheme returns the effective theme name: "default" when Theme is empty
// or unrecognized, else Theme. Pure and total, so the adapter can trust it.
func (t TUI) ResolvedTheme() string {
	if _, ok := knownThemes[t.Theme]; ok && t.Theme != "" {
		return t.Theme
	}
	return "default"
}

// Tracker holds the application-tracker settings (v1.4). Both keys default when
// absent - an existing config with no [tracker] section stays valid. Zero means
// "use the default" (see Tracker.withDefaults).
type Tracker struct {
	// GhostAfterDays is the lazy-ghost threshold: an active application whose
	// most recent activity is older than this many days is auto-ghosted on the
	// next `apply list`/`apply show`. Default 30.
	GhostAfterDays int `toml:"ghost_after_days"`

	// FollowupDefaultLagDays is the offset from today used when `apply followup`
	// omits --due. Default 7.
	FollowupDefaultLagDays int `toml:"followup_default_lag_days"`
}

// Default tracker settings (SPEC section 8).
const (
	defaultGhostAfterDays         = 30
	defaultFollowupDefaultLagDays = 7
)

// Operational defaults for keys a partial config.toml may omit. Matching the
// skeleton config: without these, a workspace config that sets only e.g.
// `[render] emit_markdown = true` would zero typst_bin ("exec: no command"),
// output_dir (PDF lands in the appdir root), and the page math.
const (
	defaultTypstBin     = "typst"
	defaultOutputDir    = "output"
	defaultPageLimit    = 1.0
	defaultPageHeightPt = 841.89 // A4
	// A parent with zero surviving children renders as a dangling heading
	// ("Languages: " with no items), so 1 is the floor, matching the shipped
	// skeleton. Absent [render.min_elements] must not mean "0 is fine".
	defaultMinElements = 1
)

// WithDefaults returns a copy of c with each operationally-required zero field
// replaced by its skeleton default. Purely presentational or behavioral toggles
// (emit_*, strict_input, score.skill_priority, ...) keep their zero values - only
// keys whose absence breaks the pipeline are defaulted. Applied once at config
// load, so every consumer sees effective values.
func (c Config) WithDefaults() Config {
	if c.Paths.TypstBin == "" {
		c.Paths.TypstBin = defaultTypstBin
	}
	if c.Paths.OutputDir == "" {
		c.Paths.OutputDir = defaultOutputDir
	}
	if c.Render.PageLimit == 0 {
		c.Render.PageLimit = defaultPageLimit
	}
	if c.Render.PageHeightPt == 0 {
		c.Render.PageHeightPt = defaultPageHeightPt
	}
	if c.Render.MinElements.JobBullets == 0 {
		c.Render.MinElements.JobBullets = defaultMinElements
	}
	if c.Render.MinElements.ProjectBullets == 0 {
		c.Render.MinElements.ProjectBullets = defaultMinElements
	}
	if c.Render.MinElements.SkillItems == 0 {
		c.Render.MinElements.SkillItems = defaultMinElements
	}
	return c
}

// WithDefaults returns a copy of t with each zero field replaced by its
// default, so callers read effective values without re-implementing the
// fallbacks.
func (t Tracker) WithDefaults() Tracker {
	if t.GhostAfterDays == 0 {
		t.GhostAfterDays = defaultGhostAfterDays
	}
	if t.FollowupDefaultLagDays == 0 {
		t.FollowupDefaultLagDays = defaultFollowupDefaultLagDays
	}
	return t
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
	// markup or disallowed URL schemes no longer fail the render - the
	// offending bullet is emitted as Typst-escaped literal text instead.
	// CLI: --force. Default false (strict).
	ForceUnsafe bool `toml:"force_unsafe"`

	// UseContainer selects the renderer backend. Valid values:
	//   "" - host typst binary (v1.0 behavior; default; byte-equiv)
	//   "false" - same as ""
	//   "true" - require a container engine; fail loud if none found
	//   "auto" - use container if engine present, else host
	// The container backend uses rootless podman if available, falling back
	// to docker. See ContainerMode for the parsed form.
	UseContainer string `toml:"use_container"`

	// StripMetadata enables an opt-in qpdf post-process that empties the
	// rendered PDF's /Author, /Creator, /Producer, /CreationDate, /ModDate.
	// Default false - existing users without qpdf installed are unaffected.
	StripMetadata bool `toml:"strip_metadata"`

	// StrictInput enables opt-in input validation at load time (section 4.2 step 1).
	// NUL bytes are rejected regardless; when true, control characters (except
	// \n and \t), invalid UTF-8, and the per-field-class byte limits below are
	// enforced too. Default false, so existing v1.0 data loads unchanged.
	StrictInput bool `toml:"strict_input"`

	// Limits holds per-field-class byte limits, enforced only when StrictInput
	// is true. Any zero field falls back to its default (see Limits.withDefaults).
	Limits Limits `toml:"limits"`

	// EmitMarkdown writes <profile>.md - the filtered resume as Markdown, meant
	// for pasting into an LLM alongside a job description. Default false, so v1.1
	// output is unchanged. Does not affect the PDF.
	EmitMarkdown bool `toml:"emit_markdown"`

	// EmitFiltered writes <profile>.filtered.toml - the post-filter resume data
	// (exactly the entities that went into the PDF). Default false. Does not
	// affect the PDF.
	EmitFiltered bool `toml:"emit_filtered"`
}

// Limits are per-field-class byte limits enforced when Render.StrictInput is
// true. A zero field means "use the default" (see withDefaults).
type Limits struct {
	Short      int `toml:"short"`       // names, titles, dates, company, location, tags
	BulletText int `toml:"bullet_text"` // bullet text and the header summary
	Notes      int `toml:"notes"`       // application notes (v1.5); reserved
	URLOrPath  int `toml:"url_or_path"` // contact hrefs and path-like fields
}

// Default per-field-class byte limits (DESIGN section 4.2 step 1).
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
