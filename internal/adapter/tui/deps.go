// Package tui is the interactive terminal front-end (v1.5). It is a pure
// adapter: every state change routes through the same internal/usecase packages
// the CLI uses, and no logic lives here that isn't just presentation. The whole
// package is gated behind the `notui` build tag - under `-tags notui` only
// deps.go and run_notui.go compile, so the charm dependencies are linked out and
// a single-binary build carries no TUI code.
package tui

import (
	"context"
	"os/exec"

	"github.com/crispuscrew/resumegen/internal/usecase/track"
)

// Deps is the injected context a TUI run needs. The cli package owns appdir and
// config resolution and constructs this; the TUI only renders and dispatches.
// It is defined without a build constraint so cli can build it under both the
// default and `notui` builds - only Run's implementation differs by tag.
//
// The function fields let the TUI reuse the exact CLI/use-case logic (render,
// prompt resolution, config layering) without duplicating any of it, keeping the
// package free of anything but presentation.
type Deps struct {
	Ctx     context.Context
	Version string
	// Theme is the resolved [tui].theme name (domain.TUI.ResolvedTheme).
	Theme string

	// Tracker is the v1.4 application-tracker use case.
	Tracker *track.Tracker
	// StaleAfterDays is the dashboard's early-warning threshold: an active
	// application with no activity for this many days is flagged as at risk.
	// cli wires it BELOW GhostAfterDays (track.StaleWarnDays) - at or above the
	// ghost threshold the warning could never fire, since List auto-ghosts
	// everything past that mark before the dashboard counts.
	StaleAfterDays int
	// GhostAfterDays is the resolved auto-ghost threshold, shown alongside the
	// warning so the badge can say when the ghost will actually happen.
	GhostAfterDays int
	// FollowupLagDays is the default followup lag when a due date is blank.
	FollowupLagDays int

	// Generate screen.
	ListProfiles func() ([]string, error)
	Render       func(ctx context.Context, profile string) (outPath string, err error)

	// Prompts screen.
	ListPrompts func(ctx context.Context) ([]PromptEntry, error)
	LoadPrompt  func(ctx context.Context, name string) (PromptForm, error)
	RunPrompt   func(ctx context.Context, name string, values map[string]string) (string, error)
	Copy        func(ctx context.Context, text string) error

	// Data screen. EditCmd builds the $EDITOR invocation for tea.ExecProcess.
	ListDataFiles func() ([]DataFile, error)
	EditCmd       func(path string) *exec.Cmd

	// Config screen.
	LoadConfig func(ctx context.Context) (ConfigView, error)
}

// PromptEntry is one row of the prompt picker.
type PromptEntry struct {
	Name        string
	Description string
	Overridden  bool
}

// PromptField describes one input a template needs the user to supply.
type PromptField struct {
	Key      string
	Source   string
	Flag     string
	Field    string
	Default  string
	Required bool
}

// PromptForm is a template's identity plus the fields to collect. Fields is in a
// stable order (sorted by key) so the form is deterministic.
type PromptForm struct {
	Name        string
	Description string
	Fields      []PromptField
}

// DataFile is one editable data file in the appdir.
type DataFile struct {
	Name string
	Path string
}

// ConfigLine is one effective key/value pair for the read-only config view.
type ConfigLine struct {
	Key   string
	Value string
}

// ConfigView is the resolved config plus where it came from.
type ConfigView struct {
	Appdir string
	Origin string
	Lines  []ConfigLine
}
