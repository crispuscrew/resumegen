//go:build !notui

package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase/track"
)

const dateLayout = "2006-01-02"

// detailMode is the sub-state of the detail screen: either viewing, or capturing
// input for one of the mutating actions.
type detailMode int

const (
	detailView detailMode = iota
	detailStatusPick
	detailNote
	detailFollowupAction
	detailFollowupDue
	detailFollowupDone
)

type detailState struct {
	app        domain.Application
	mode       detailMode
	targets    []domain.Status // transition choices in detailStatusPick
	statusCur  int
	input      textinput.Model // reused for note / followup action / due
	pendingAct string          // followup action captured before the due prompt
	flash      string          // transient success message
	errMsg     string          // transient error message
	busy       bool            // a mutating command is in flight; ignore re-submits
}

// capturing reports whether the detail screen is in a text-entry mode, so the
// root can hand keys straight to the input instead of treating "q"/"?" globally.
func (d detailState) capturing() bool {
	return d.mode == detailNote || d.mode == detailFollowupAction ||
		d.mode == detailFollowupDue || d.mode == detailFollowupDone
}

func newTextInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 512
	ti.Width = 48
	// A static (non-blinking) cursor keeps the input free of timer commands, so
	// the model stays deterministic and cheap to drive in tests.
	ti.Cursor.SetMode(cursor.CursorStatic)
	ti.Focus() // with a static cursor this returns nil; input still works
	return ti
}

// resolveDue parses a blank-or-YYYY-MM-DD due string. Blank maps to the zero
// time: track.AddFollowup owns the "today + configured lag" rule.
func (m model) resolveDue(val string) (time.Time, error) {
	val = strings.TrimSpace(val)
	if val == "" {
		return time.Time{}, nil
	}
	d, err := time.Parse(dateLayout, val)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: want YYYY-MM-DD", val)
	}
	return d, nil
}

func (m model) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, isKey := msg.(tea.KeyMsg)
	d := &m.detail
	switch d.mode {
	case detailView:
		if !isKey {
			return m, nil
		}
		switch km.String() {
		case "esc":
			m = m.switchTo(screenList)
			return m, nil
		case "s":
			d.targets = track.ManualTargets(d.app.Status)
			d.flash, d.errMsg = "", ""
			if len(d.targets) == 0 {
				d.errMsg = "no transitions from " + string(d.app.Status)
				return m, nil
			}
			d.statusCur = 0
			d.mode = detailStatusPick
			return m, nil
		case "n":
			d.input = newTextInput("note text")
			d.flash, d.errMsg = "", ""
			d.mode = detailNote
			return m, nil
		case "f":
			d.input = newTextInput("what to do")
			d.flash, d.errMsg = "", ""
			d.mode = detailFollowupAction
			return m, nil
		case "d":
			if len(d.app.Followups) == 0 {
				d.errMsg = "no followups to complete"
				return m, nil
			}
			d.input = newTextInput("followup # to mark done (as listed)")
			d.flash, d.errMsg = "", ""
			d.mode = detailFollowupDone
			return m, nil
		}
		return m, nil

	case detailStatusPick:
		if !isKey {
			return m, nil
		}
		switch km.String() {
		case "esc":
			// esc abandons the pending action, so busy must be cleared with the
			// mode: leaving it set makes the re-entrancy guard silently reject
			// every subsequent submit on this screen, with no feedback.
			d.mode = detailView
			d.busy = false
			return m, nil
		case "up", "k":
			if d.statusCur > 0 {
				d.statusCur--
			}
			return m, nil
		case "down", "j":
			if d.statusCur < len(d.targets)-1 {
				d.statusCur++
			}
			return m, nil
		case "enter":
			if d.busy { // key auto-repeat or a double-tap must not dispatch twice
				return m, nil
			}
			d.busy = true
			return m, m.transitionCmd(d.app.ID, d.targets[d.statusCur])
		}
		return m, nil

	case detailNote:
		if isKey {
			switch km.String() {
			case "esc":
				d.mode = detailView
				d.busy = false
				return m, nil
			case "enter":
				if d.busy {
					return m, nil
				}
				val := strings.TrimSpace(d.input.Value())
				if val == "" {
					d.mode = detailView
					return m, nil
				}
				d.busy = true
				return m, m.noteCmd(d.app.ID, val)
			}
		}
		var cmd tea.Cmd
		d.input, cmd = d.input.Update(msg)
		return m, cmd

	case detailFollowupAction:
		if isKey {
			switch km.String() {
			case "esc":
				d.mode = detailView
				d.busy = false
				return m, nil
			case "enter":
				val := strings.TrimSpace(d.input.Value())
				if val == "" {
					d.mode = detailView
					return m, nil
				}
				d.pendingAct = val
				d.input = newTextInput("due YYYY-MM-DD (blank = default lag)")
				d.mode = detailFollowupDue
				return m, nil
			}
		}
		var cmd tea.Cmd
		d.input, cmd = d.input.Update(msg)
		return m, cmd

	case detailFollowupDue:
		if isKey {
			switch km.String() {
			case "esc":
				d.mode = detailView
				d.busy = false
				return m, nil
			case "enter":
				if d.busy {
					return m, nil
				}
				due, err := m.resolveDue(d.input.Value())
				if err != nil {
					d.errMsg = err.Error()
					d.mode = detailView
					return m, nil
				}
				d.busy = true
				return m, m.followupCmd(d.app.ID, due, d.pendingAct)
			}
		}
		var cmd tea.Cmd
		d.input, cmd = d.input.Update(msg)
		return m, cmd

	case detailFollowupDone:
		if isKey {
			switch km.String() {
			case "esc":
				d.mode = detailView
				d.busy = false
				return m, nil
			case "enter":
				if d.busy {
					return m, nil
				}
				n, err := strconv.Atoi(strings.TrimSpace(d.input.Value()))
				if err != nil || n < 1 {
					d.errMsg = "enter the followup number as listed (1, 2, ...)"
					d.mode = detailView
					return m, nil
				}
				d.busy = true
				return m, m.completeFollowupCmd(d.app.ID, n)
			}
		}
		var cmd tea.Cmd
		d.input, cmd = d.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

// --- views ---

func (m model) listView() string {
	if m.loading {
		return m.styl.subtle.Render("loading…")
	}
	if m.loadErr != nil {
		return m.styl.errText.Render("error: " + m.loadErr.Error())
	}
	if len(m.apps) == 0 {
		return m.styl.subtle.Render("no applications yet — press n to create one.")
	}
	vis := m.visibleApps()
	var b strings.Builder
	if m.filtering {
		b.WriteString(m.styl.label.Render("filter ") + m.filterInput.View() + "\n")
	} else if m.filter != "" {
		b.WriteString(m.styl.subtle.Render("filter: "+m.filter+"  (esc clears)") + "\n")
	}
	if m.filter != "" || m.filtering {
		b.WriteString(m.styl.subtle.Render(fmt.Sprintf("%d of %d", len(vis), len(m.apps))) + "\n\n")
	}
	if len(vis) == 0 {
		b.WriteString(m.styl.subtle.Render("no matches") + "\n")
	}
	for i, a := range vis {
		line := fmt.Sprintf("%-24s  %-16s  %-18s  %s", truncate(a.Company, 24), truncate(a.Role, 16), a.ID, a.Status)
		if i == m.cursor {
			b.WriteString(m.styl.selected.Render("› "+line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}
	if m.flash != "" {
		b.WriteString("\n" + m.styl.flash.Render("✓ "+m.flash) + "\n")
	}
	return b.String()
}

func (m model) detailView() string {
	d := m.detail
	a := d.app
	var b strings.Builder

	b.WriteString(m.styl.title.Render(a.Company+" — "+a.Role) + "\n")
	b.WriteString(m.styl.subtle.Render(a.ID) + "\n\n")

	b.WriteString(m.field("Status", string(a.Status)))
	if a.Profile != "" {
		b.WriteString(m.field("Profile", a.Profile))
	}
	if a.Source != "" {
		b.WriteString(m.field("Source", a.Source))
	}
	if !a.AppliedAt.IsZero() {
		b.WriteString(m.field("Applied", a.AppliedAt.Format(dateLayout)))
	}
	if a.JDPath != "" {
		b.WriteString(m.field("JD path", a.JDPath))
	}
	if a.SalaryRange != "" {
		b.WriteString(m.field("Salary", a.SalaryRange))
	}
	if a.Notes != "" {
		b.WriteString(m.field("Notes", a.Notes))
	}

	if len(a.Followups) > 0 {
		b.WriteString("\n" + m.styl.label.Render("Followups") + "\n")
		for i, f := range a.Followups {
			mark := " "
			if f.Done {
				mark = "x"
			}
			fmt.Fprintf(&b, "  %d. [%s] %s  %s\n", i+1, mark, f.Due.Format(dateLayout), f.Action)
		}
	}
	if len(a.Contacts) > 0 {
		b.WriteString("\n" + m.styl.label.Render("Contacts") + "\n")
		for _, c := range a.Contacts {
			line := c.Name
			if c.Role != "" {
				line += " (" + c.Role + ")"
			}
			if c.Channel != "" {
				line += " via " + c.Channel
			}
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}
	if len(a.Events) > 0 {
		b.WriteString("\n" + m.styl.label.Render("Events") + "\n")
		for _, e := range a.Events {
			note := ""
			if e.Note != "" {
				note = "  " + e.Note
			}
			fmt.Fprintf(&b, "  %s  %-8s%s\n", e.At.Format(dateLayout), e.Kind, note)
		}
	}

	// interactive overlays / flashes
	switch d.mode {
	case detailStatusPick:
		b.WriteString("\n" + m.styl.label.Render("Change status to:") + "\n")
		for i, t := range d.targets {
			if i == d.statusCur {
				b.WriteString(m.styl.selected.Render("› "+string(t)) + "\n")
			} else {
				b.WriteString("  " + string(t) + "\n")
			}
		}
	case detailNote:
		b.WriteString("\n" + m.styl.label.Render("Add note") + "\n  " + d.input.View() + "\n")
	case detailFollowupAction:
		b.WriteString("\n" + m.styl.label.Render("Add followup — action") + "\n  " + d.input.View() + "\n")
	case detailFollowupDue:
		b.WriteString("\n" + m.styl.label.Render("Add followup — due date") + "\n  " + d.input.View() + "\n")
	case detailFollowupDone:
		b.WriteString("\n" + m.styl.label.Render("Complete followup — number") + "\n  " + d.input.View() + "\n")
	}
	if d.flash != "" {
		b.WriteString("\n" + m.styl.flash.Render("✓ "+d.flash) + "\n")
	}
	if d.errMsg != "" {
		b.WriteString("\n" + m.styl.errText.Render("✗ "+d.errMsg) + "\n")
	}
	return b.String()
}

func (m model) detailFooter() string {
	switch m.detail.mode {
	case detailStatusPick:
		return "↑/↓: choose · enter: apply · esc: cancel"
	case detailNote, detailFollowupAction, detailFollowupDue, detailFollowupDone:
		return "type · enter: submit · esc: cancel"
	default:
		return "s: status · n: note · f: followup · d: followup done · esc: back · q: quit"
	}
}

// truncate shortens s to at most n runes (never slicing mid-rune, so Cyrillic
// and other multi-byte text stays valid), ending with an ellipsis when cut.
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
