//go:build !notui

package tui

import tea "github.com/charmbracelet/bubbletea"

// Supported reports whether this build includes the TUI (false under -tags notui).
func Supported() bool { return true }

// Run starts the interactive TUI over the alt screen and blocks until the user
// quits. It builds the root model from the injected Deps; all IO happens through
// the use cases those Deps carry.
func Run(d Deps) error {
	m := model{
		deps:    d,
		styl:    newStyles(d.Theme),
		active:  screenDashboard,
		loading: true,
	}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
