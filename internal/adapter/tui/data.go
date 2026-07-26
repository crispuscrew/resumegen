//go:build !notui

package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type dataState struct {
	files  []DataFile
	cursor int
	loaded bool
	err    error
	flash  string
}

type dataFilesMsg struct {
	files []DataFile
	err   error
}

type editorFinishedMsg struct{ err error }

func (m model) loadDataFiles() tea.Cmd {
	list := m.deps.ListDataFiles
	return func() tea.Msg {
		if list == nil {
			return dataFilesMsg{err: errNoCapability}
		}
		files, err := list()
		return dataFilesMsg{files: files, err: err}
	}
}

func (m model) onDataFiles(msg dataFilesMsg) (tea.Model, tea.Cmd) {
	m.data.loaded = true
	m.data.files = msg.files
	m.data.err = msg.err
	if m.data.cursor >= len(m.data.files) {
		m.data.cursor = maxInt(0, len(m.data.files)-1)
	}
	return m, nil
}

func (m model) onEditorFinished(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.data.flash = "editor error: " + msg.err.Error()
	} else {
		m.data.flash = "editor closed"
	}
	return m, nil
}

func (m model) updateData(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m = m.switchTo(screenDashboard)
		return m, nil
	case "up", "k":
		if m.data.cursor > 0 {
			m.data.cursor--
		}
		return m, nil
	case "down", "j":
		if m.data.cursor < len(m.data.files)-1 {
			m.data.cursor++
		}
		return m, nil
	case "r":
		m.data.loaded = false
		return m, m.loadDataFiles()
	case "enter":
		if len(m.data.files) == 0 || m.deps.EditCmd == nil {
			return m, nil
		}
		path := m.data.files[m.data.cursor].Path
		// tea.ExecProcess suspends the TUI, runs $EDITOR against the file on the
		// real terminal, and resumes with editorFinishedMsg.
		return m, tea.ExecProcess(m.deps.EditCmd(path), func(err error) tea.Msg {
			return editorFinishedMsg{err: err}
		})
	}
	return m, nil
}

func (m model) dataView() string {
	d := m.data
	if !d.loaded {
		return m.styl.subtle.Render("loading…")
	}
	if d.err != nil {
		return m.styl.errText.Render("error: " + d.err.Error())
	}
	if len(d.files) == 0 {
		return m.styl.subtle.Render("no data files (data/*.toml) in this appdir — run `resumegen init` first.")
	}
	var b strings.Builder
	b.WriteString(m.styl.label.Render("Data files") + "  " + m.styl.subtle.Render("(enter opens $EDITOR)") + "\n\n")
	for i, f := range d.files {
		if i == d.cursor {
			b.WriteString(m.styl.selected.Render("› "+f.Name) + "\n")
		} else {
			b.WriteString("  " + f.Name + "\n")
		}
	}
	if d.flash != "" {
		b.WriteString("\n" + m.styl.flash.Render("✓ "+d.flash) + "\n")
	}
	return b.String()
}
