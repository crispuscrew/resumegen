//go:build !notui

package tui

import "github.com/charmbracelet/lipgloss"

// styles is a resolved theme palette. lipgloss/termenv degrade automatically on
// dumb terminals and honor NO_COLOR, so no manual gating is needed.
type styles struct {
	title    lipgloss.Style
	subtle   lipgloss.Style
	selected lipgloss.Style
	label    lipgloss.Style
	flash    lipgloss.Style
	errText  lipgloss.Style
	footer   lipgloss.Style
	badge    lipgloss.Style
}

// themes is the palette registry: adding a theme is one entry here plus its
// name in domain.knownThemes. newStyles falls back to default for anything
// unknown, so a stale config can never crash the TUI.
var themes = map[string]func() styles{
	"default": defaultStyles,
}

// newStyles builds the palette for a resolved theme name.
func newStyles(theme string) styles {
	if f, ok := themes[theme]; ok {
		return f()
	}
	return defaultStyles()
}

func defaultStyles() styles {
	accent := lipgloss.Color("62") // muted purple
	dim := lipgloss.Color("241")   // grey
	good := lipgloss.Color("35")   // green
	warn := lipgloss.Color("208")  // orange
	return styles{
		title:    lipgloss.NewStyle().Bold(true).Foreground(accent),
		subtle:   lipgloss.NewStyle().Foreground(dim),
		selected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(accent),
		label:    lipgloss.NewStyle().Bold(true).Foreground(good),
		flash:    lipgloss.NewStyle().Foreground(good),
		errText:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196")),
		footer:   lipgloss.NewStyle().Foreground(dim),
		badge:    lipgloss.NewStyle().Foreground(warn),
	}
}
