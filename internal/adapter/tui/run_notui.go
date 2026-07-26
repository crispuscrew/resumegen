//go:build notui

package tui

import "errors"

// Supported reports whether this build includes the TUI (false under -tags notui).
func Supported() bool { return false }

// Run is the no-TUI stub. Under `-tags notui` the bubbletea/bubbles/lipgloss
// packages are never imported, so the binary links zero charm code; `resumegen
// tui` reports that support was compiled out and points back at the CLI.
func Run(_ Deps) error {
	return errors.New("this build was compiled without TUI support (-tags notui); use the resumegen CLI subcommands (apply/prompt/render) instead")
}
