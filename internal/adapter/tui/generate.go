//go:build !notui

package tui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type genPhase int

const (
	genIdle genPhase = iota
	genRunning
	genDone
	genError
)

type generateState struct {
	profiles []string
	cursor   int
	loaded   bool
	loadErr  error

	phase   genPhase
	outPath string
	errMsg  string
	cancel  context.CancelFunc // non-nil while a render is in flight
}

type profilesLoadedMsg struct {
	profiles []string
	err      error
}

type renderDoneMsg struct {
	outPath string
	err     error
}

func (m model) loadProfiles() tea.Cmd {
	list := m.deps.ListProfiles
	return func() tea.Msg {
		if list == nil {
			return profilesLoadedMsg{err: errNoCapability}
		}
		p, err := list()
		return profilesLoadedMsg{profiles: p, err: err}
	}
}

func (m model) onProfilesLoaded(msg profilesLoadedMsg) (tea.Model, tea.Cmd) {
	m.generate.loaded = true
	m.generate.profiles = msg.profiles
	m.generate.loadErr = msg.err
	if m.generate.cursor >= len(m.generate.profiles) {
		m.generate.cursor = maxInt(0, len(m.generate.profiles)-1)
	}
	return m, nil
}

// renderCmd runs the injected render under ctx (cancellable) in a goroutine, so
// the UI stays responsive and esc can kill the typst subprocess mid-render.
func (m model) renderCmd(ctx context.Context, profile string) tea.Cmd {
	render := m.deps.Render
	return func() tea.Msg {
		if render == nil {
			return renderDoneMsg{err: errNoCapability}
		}
		out, err := render(ctx, profile)
		return renderDoneMsg{outPath: out, err: err}
	}
}

func (m model) onRenderDone(msg renderDoneMsg) (tea.Model, tea.Cmd) {
	g := &m.generate
	if g.cancel != nil {
		g.cancel() // release the context regardless of outcome
		g.cancel = nil
	}
	if msg.err != nil {
		g.phase = genError
		g.errMsg = msg.err.Error()
		return m, nil
	}
	g.phase = genDone
	g.outPath = msg.outPath
	return m, nil
}

func (m model) updateGenerate(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	g := &m.generate
	switch msg.String() {
	case "esc":
		if g.phase == genRunning {
			if g.cancel != nil {
				g.cancel() // cancel the in-flight render; renderDoneMsg will follow
			}
			return m, nil
		}
		m = m.switchTo(screenDashboard)
		return m, nil
	case "up", "k":
		if g.phase != genRunning && g.cursor > 0 {
			g.cursor--
		}
		return m, nil
	case "down", "j":
		if g.phase != genRunning && g.cursor < len(g.profiles)-1 {
			g.cursor++
		}
		return m, nil
	case "r":
		if g.phase != genRunning {
			g.loaded = false
			return m, m.loadProfiles()
		}
		return m, nil
	case "enter":
		if g.phase == genRunning || len(g.profiles) == 0 {
			return m, nil
		}
		profile := g.profiles[g.cursor]
		ctx, cancel := context.WithCancel(m.deps.Ctx)
		g.cancel = cancel
		g.phase = genRunning
		g.outPath = ""
		g.errMsg = ""
		return m, m.renderCmd(ctx, profile)
	}
	return m, nil
}

func (m model) generateView() string {
	g := m.generate
	if !g.loaded {
		return m.styl.subtle.Render("loading…")
	}
	if g.loadErr != nil {
		return m.styl.errText.Render("error: " + g.loadErr.Error())
	}
	if len(g.profiles) == 0 {
		return m.styl.subtle.Render("no profiles (profiles/*.toml) in this appdir — run `resumegen init` first.")
	}
	var b strings.Builder
	b.WriteString(m.styl.label.Render("Profiles") + "\n\n")
	for i, p := range g.profiles {
		if i == g.cursor {
			b.WriteString(m.styl.selected.Render("› "+p) + "\n")
		} else {
			b.WriteString("  " + p + "\n")
		}
	}
	b.WriteString("\n")
	switch g.phase {
	case genRunning:
		b.WriteString(m.styl.badge.Render("● rendering… (esc cancels)") + "\n")
	case genDone:
		b.WriteString(m.styl.flash.Render("✓ rendered → "+g.outPath) + "\n")
	case genError:
		b.WriteString(m.styl.errText.Render("✗ "+g.errMsg) + "\n")
	}
	return b.String()
}

func (m model) generateFooter() string {
	if m.generate.phase == genRunning {
		return "esc: cancel render · ctrl+c: quit"
	}
	return "↑/↓: pick · enter: render · r: reload · esc: dashboard · 1-6: switch · q: quit"
}
