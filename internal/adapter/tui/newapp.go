//go:build !notui

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase/track"
)

// newAppFields is the creation form, in display order. The indices are relied
// on when assembling track.NewInput in updateNewApp.
var newAppFields = []struct {
	key         string
	placeholder string
}{
	{"company", "hiring company (required)"},
	{"role", "role / title (required)"},
	{"profile", "resume profile used"},
	{"source", "where the JD came from (hh.ru, referral, …)"},
	{"jd", "path to a JD text file you keep (stored, never fetched)"},
	{"salary", "salary range (free text)"},
}

// newAppState is the `n` creation form: one text input per field. The entry is
// created as drafting (like `apply new` without --status); the user advances it
// with `s` from the detail view that opens on success.
type newAppState struct {
	inputs   []textinput.Model
	fieldCur int
	errMsg   string
	busy     bool // a create is in flight; ignore re-submits
}

func newNewAppState() newAppState {
	s := newAppState{}
	for i, f := range newAppFields {
		ti := newTextInput(f.placeholder)
		if i != 0 {
			ti.Blur()
		}
		s.inputs = append(s.inputs, ti)
	}
	return s
}

func (s *newAppState) focusField(delta int) {
	s.inputs[s.fieldCur].Blur()
	s.fieldCur = (s.fieldCur + delta + len(s.inputs)) % len(s.inputs)
	s.inputs[s.fieldCur].Focus()
}

// --- message / command ---

type newAppDoneMsg struct {
	app domain.Application
	err error
}

func (m model) createAppCmd(in track.NewInput) tea.Cmd {
	ctx, tr := m.deps.Ctx, m.deps.Tracker
	return func() tea.Msg {
		app, err := tr.New(ctx, in)
		return newAppDoneMsg{app: app, err: err}
	}
}

// openNewApp resets the form and switches to it (from the dashboard or list).
func (m model) openNewApp() (tea.Model, tea.Cmd) {
	m.newapp = newNewAppState()
	m = m.switchTo(screenNewApp)
	return m, nil
}

// --- update ---

func (m model) updateNewApp(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, isKey := msg.(tea.KeyMsg)
	s := &m.newapp
	if isKey {
		switch km.String() {
		case "esc":
			m = m.switchTo(screenList)
			return m, nil
		case "tab", "down":
			s.focusField(1)
			return m, nil
		case "shift+tab", "up":
			s.focusField(-1)
			return m, nil
		case "enter":
			if s.busy {
				return m, nil
			}
			if s.fieldCur < len(s.inputs)-1 {
				s.focusField(1) // enter walks the form; submit happens on the last field
				return m, nil
			}
			in := track.NewInput{
				Company:     strings.TrimSpace(s.inputs[0].Value()),
				Role:        strings.TrimSpace(s.inputs[1].Value()),
				Profile:     strings.TrimSpace(s.inputs[2].Value()),
				Source:      strings.TrimSpace(s.inputs[3].Value()),
				JDPath:      strings.TrimSpace(s.inputs[4].Value()),
				SalaryRange: strings.TrimSpace(s.inputs[5].Value()),
			}
			s.busy = true
			s.errMsg = ""
			return m, m.createAppCmd(in)
		}
	}
	var cmd tea.Cmd
	s.inputs[s.fieldCur], cmd = s.inputs[s.fieldCur].Update(msg)
	return m, cmd
}

func (m model) onNewAppDone(msg newAppDoneMsg) (tea.Model, tea.Cmd) {
	m.newapp.busy = false
	if msg.err != nil {
		// Validation errors (missing company/role) keep the form open to fix.
		if m.active == screenNewApp {
			m.newapp.errMsg = msg.err.Error()
		}
		return m, nil
	}
	if m.active != screenNewApp {
		// The user esc'd before the result landed; the entry exists on disk —
		// just refresh the list rather than yanking them into the detail view.
		return m, m.loadApps()
	}
	m.detail = detailState{app: msg.app, mode: detailView, flash: "created " + msg.app.ID}
	m = m.switchTo(screenDetail)
	return m, m.loadApps()
}

// --- view ---

func (m model) newAppView() string {
	s := m.newapp
	var b strings.Builder
	b.WriteString(m.styl.title.Render("New application") + "\n\n")
	for i := range s.inputs {
		cur := "  "
		if i == s.fieldCur {
			cur = m.styl.selected.Render("› ")
		}
		b.WriteString(cur + m.styl.subtle.Render(fmt.Sprintf("%-9s", newAppFields[i].key)) + " " + s.inputs[i].View() + "\n")
	}
	b.WriteString("\n" + m.styl.subtle.Render("created as drafting — press s on the new entry to advance its status") + "\n")
	if s.busy {
		b.WriteString("\n" + m.styl.badge.Render("● creating…") + "\n")
	}
	if s.errMsg != "" {
		b.WriteString("\n" + m.styl.errText.Render("✗ "+s.errMsg) + "\n")
	}
	return b.String()
}

func (m model) newAppFooter() string {
	return "tab: next field · enter: next / create · esc: cancel"
}
