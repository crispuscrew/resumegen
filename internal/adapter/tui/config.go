//go:build !notui

package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type configState struct {
	view   ConfigView
	loaded bool
	err    error
}

type configLoadedMsg struct {
	view ConfigView
	err  error
}

func (m model) loadConfig() tea.Cmd {
	ctx, load := m.deps.Ctx, m.deps.LoadConfig
	return func() tea.Msg {
		if load == nil {
			return configLoadedMsg{err: errNoCapability}
		}
		v, err := load(ctx)
		return configLoadedMsg{view: v, err: err}
	}
}

func (m model) onConfigLoaded(msg configLoadedMsg) (tea.Model, tea.Cmd) {
	m.config.loaded = true
	m.config.view = msg.view
	m.config.err = msg.err
	return m, nil
}

func (m model) updateConfig(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m = m.switchTo(screenDashboard)
		return m, nil
	case "r":
		m.config.loaded = false
		return m, m.loadConfig()
	}
	return m, nil
}

func (m model) configView() string {
	c := m.config
	if !c.loaded {
		return m.styl.subtle.Render("loading…")
	}
	if c.err != nil {
		return m.styl.errText.Render("error: " + c.err.Error())
	}
	var b strings.Builder
	b.WriteString(m.field("appdir", c.view.Appdir))
	b.WriteString(m.field("origin", c.view.Origin))
	b.WriteString("\n" + m.styl.label.Render("Effective config") + "\n")
	for _, ln := range c.view.Lines {
		b.WriteString(m.fieldWide(ln.Key, ln.Value))
	}
	b.WriteString("\n" + m.styl.subtle.Render("read-only - edit config.toml in your appdir to change these") + "\n")
	return b.String()
}
