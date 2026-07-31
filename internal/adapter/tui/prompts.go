//go:build !notui

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type promptsPhase int

const (
	promptsPickList promptsPhase = iota
	promptsFill
	promptsResult
)

type promptsState struct {
	entries []PromptEntry
	cursor  int
	loaded  bool
	loadErr error

	phase     promptsPhase
	form      PromptForm
	inputs    []textinput.Model // one per collected field
	fieldKeys []string          // value-map key per input (parallel to inputs)
	fieldCur  int
	result    string
	errMsg    string
}

// capturing is true while the user is typing into the fill form, so global
// navigation keys (digits, q, ?) are handed to the focused field instead. A
// zero-input form (auto-run in flight) captures nothing — nav keys stay live.
func (p promptsState) capturing() bool { return p.phase == promptsFill && len(p.inputs) > 0 }

func (p *promptsState) focusField(delta int) {
	if len(p.inputs) == 0 {
		return
	}
	p.inputs[p.fieldCur].Blur()
	p.fieldCur = (p.fieldCur + delta + len(p.inputs)) % len(p.inputs)
	p.inputs[p.fieldCur].Focus()
}

// buildPromptInputs turns a form into the text fields the user must fill. Sources
// that resolve from context (data-dump, app-id) don't get a per-field input;
// instead a single "__profile"/"__app" field is added when any such input exists.
func buildPromptInputs(form PromptForm) ([]textinput.Model, []string) {
	var inputs []textinput.Model
	var keys []string
	needProfile, needApp := false, false
	add := func(key, placeholder, def string) {
		ti := newTextInput(placeholder)
		if def != "" {
			ti.SetValue(def)
		}
		inputs = append(inputs, ti)
		keys = append(keys, key)
	}
	for _, f := range form.Fields {
		switch f.Source {
		case "flag", "prompt", "stdin":
			add(f.Key, fieldPlaceholder(f), f.Default)
		case "jd-file":
			// The field holds a PATH; a spec default is fallback CONTENT (CLI
			// semantics), so it must not be pre-filled here — leaving the field
			// empty lets resolveFromValues apply the default as content.
			add(f.Key, "path to a "+f.Key+" file", "")
		case "data-dump":
			needProfile = true
		case "app-id":
			needApp = true
		}
	}
	if needProfile {
		add("__profile", "resume profile for the data-dump", "default")
	}
	if needApp {
		add("__app", "application id (from the tracker)", "")
	}
	if len(inputs) > 0 {
		inputs[0].Focus()
		for i := 1; i < len(inputs); i++ {
			inputs[i].Blur()
		}
	}
	return inputs, keys
}

func fieldPlaceholder(f PromptField) string {
	if f.Required {
		return f.Key + " (required)"
	}
	return f.Key
}

// --- messages / commands ---

type promptsLoadedMsg struct {
	entries []PromptEntry
	err     error
}

type promptFormMsg struct {
	form PromptForm
	err  error
}

type promptRunMsg struct {
	text string
	err  error
}

type copyDoneMsg struct {
	err   error
	label string // success message (defaults to "copied to clipboard")
}

func (m model) loadPrompts() tea.Cmd {
	ctx, list := m.deps.Ctx, m.deps.ListPrompts
	return func() tea.Msg {
		if list == nil {
			return promptsLoadedMsg{err: errNoCapability}
		}
		e, err := list(ctx)
		return promptsLoadedMsg{entries: e, err: err}
	}
}

func (m model) loadPromptForm(name string) tea.Cmd {
	ctx, load := m.deps.Ctx, m.deps.LoadPrompt
	return func() tea.Msg {
		if load == nil {
			return promptFormMsg{err: errNoCapability}
		}
		form, err := load(ctx, name)
		return promptFormMsg{form: form, err: err}
	}
}

func (m model) runPromptCmd() tea.Cmd {
	p := m.prompts
	values := make(map[string]string, len(p.fieldKeys))
	for i, key := range p.fieldKeys {
		values[key] = p.inputs[i].Value()
	}
	name := p.form.Name
	ctx, run := m.deps.Ctx, m.deps.RunPrompt
	return func() tea.Msg {
		if run == nil {
			return promptRunMsg{err: errNoCapability}
		}
		text, err := run(ctx, name, values)
		return promptRunMsg{text: text, err: err}
	}
}

func (m model) copyCmd(text, label string) tea.Cmd {
	ctx, cp := m.deps.Ctx, m.deps.Copy
	return func() tea.Msg {
		if cp == nil {
			return copyDoneMsg{err: errNoCapability}
		}
		return copyDoneMsg{err: cp(ctx, text), label: label}
	}
}

func (m model) onPromptsLoaded(msg promptsLoadedMsg) (tea.Model, tea.Cmd) {
	m.prompts.loaded = true
	m.prompts.entries = msg.entries
	m.prompts.loadErr = msg.err
	if m.prompts.cursor >= len(m.prompts.entries) {
		m.prompts.cursor = maxInt(0, len(m.prompts.entries)-1)
	}
	return m, nil
}

func (m model) onPromptForm(msg promptFormMsg) (tea.Model, tea.Cmd) {
	p := &m.prompts
	// Stale delivery guard: only the picker awaits a form. A second enter on the
	// picker, or a form arriving after esc/navigation, must not wipe live state.
	if m.active != screenPrompts || p.phase != promptsPickList {
		return m, nil
	}
	if msg.err != nil {
		p.errMsg = msg.err.Error()
		return m, nil
	}
	p.form = msg.form
	p.inputs, p.fieldKeys = buildPromptInputs(msg.form)
	p.fieldCur = 0
	p.errMsg = ""
	p.phase = promptsFill
	if len(p.inputs) == 0 {
		// Nothing to fill (e.g. a single data-dump with a default profile): run now.
		return m, m.runPromptCmd()
	}
	return m, nil
}

func (m model) onPromptRun(msg promptRunMsg) (tea.Model, tea.Cmd) {
	p := &m.prompts
	// Stale delivery guard: a run belongs to the fill form that requested it. If
	// the user esc'd back to the picker or left the screen, drop the result.
	if m.active != screenPrompts || p.phase != promptsFill {
		return m, nil
	}
	if msg.err != nil {
		p.errMsg = msg.err.Error() // stay on the fill form so the user can fix inputs
		return m, nil
	}
	p.result = msg.text
	m.flash = ""
	p.phase = promptsResult
	return m, nil
}

func (m model) onCopyDone(msg copyDoneMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.err != nil:
		m.flash = "copy failed: " + msg.err.Error()
	case msg.label != "":
		m.flash = msg.label
	default:
		m.flash = "copied to clipboard"
	}
	return m, nil
}

// --- update ---

func (m model) updatePrompts(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.prompts.phase {
	case promptsPickList:
		return m.updatePromptsPick(msg)
	case promptsFill:
		return m.updatePromptsFill(msg)
	case promptsResult:
		return m.updatePromptsResult(msg)
	}
	return m, nil
}

func (m model) updatePromptsPick(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := &m.prompts
	switch msg.String() {
	case "esc":
		m = m.switchTo(screenDashboard)
		return m, nil
	case "up", "k":
		if p.cursor > 0 {
			p.cursor--
		}
		return m, nil
	case "down", "j":
		if p.cursor < len(p.entries)-1 {
			p.cursor++
		}
		return m, nil
	case "r":
		p.loaded = false
		return m, m.loadPrompts()
	case "enter":
		if len(p.entries) == 0 {
			return m, nil
		}
		return m, m.loadPromptForm(p.entries[p.cursor].Name)
	}
	return m, nil
}

func (m model) updatePromptsFill(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := &m.prompts
	switch msg.String() {
	case "esc":
		p.phase = promptsPickList
		p.inputs, p.fieldKeys, p.errMsg = nil, nil, ""
		return m, nil
	case "tab", "down":
		p.focusField(1)
		return m, nil
	case "shift+tab", "up":
		p.focusField(-1)
		return m, nil
	case "enter":
		if len(p.inputs) == 0 {
			return m, nil // zero-input auto-run already in flight; don't re-run
		}
		if p.fieldCur < len(p.inputs)-1 {
			p.focusField(1)
			return m, nil
		}
		return m, m.runPromptCmd() // enter on the last field runs
	}
	if len(p.inputs) > 0 {
		var cmd tea.Cmd
		p.inputs[p.fieldCur], cmd = p.inputs[p.fieldCur].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) updatePromptsResult(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := &m.prompts
	switch msg.String() {
	case "esc":
		p.phase = promptsPickList
		p.result = ""
		return m, nil
	case "y":
		return m, m.copyCmd(p.result, "copied to clipboard")
	}
	return m, nil
}

// --- view ---

func (m model) promptsView() string {
	p := m.prompts
	switch p.phase {
	case promptsFill:
		return m.promptsFillView(p)
	case promptsResult:
		return m.promptsResultView(p)
	default:
		return m.promptsPickView(p)
	}
}

func (m model) promptsPickView(p promptsState) string {
	if !p.loaded {
		return m.styl.subtle.Render("loading…")
	}
	if p.loadErr != nil {
		return m.styl.errText.Render("error: " + p.loadErr.Error())
	}
	if len(p.entries) == 0 {
		return m.styl.subtle.Render("no prompt templates found.")
	}
	var b strings.Builder
	b.WriteString(m.styl.label.Render("Prompt templates") + "\n\n")
	for i, e := range p.entries {
		marker := ""
		if e.Overridden {
			marker = m.styl.badge.Render(" (custom)")
		}
		if i == p.cursor {
			b.WriteString(m.styl.selected.Render("› "+e.Name) + marker + "\n")
			if e.Description != "" {
				b.WriteString("    " + m.styl.subtle.Render(e.Description) + "\n")
			}
		} else {
			b.WriteString("  " + e.Name + marker + "\n")
		}
	}
	if p.errMsg != "" {
		b.WriteString("\n" + m.styl.errText.Render("✗ "+p.errMsg) + "\n")
	}
	return b.String()
}

func (m model) promptsFillView(p promptsState) string {
	var b strings.Builder
	b.WriteString(m.styl.title.Render(p.form.Name) + "\n")
	if p.form.Description != "" {
		b.WriteString(m.styl.subtle.Render(p.form.Description) + "\n")
	}
	b.WriteString("\n")
	if len(p.inputs) == 0 {
		b.WriteString(m.styl.subtle.Render("no inputs needed — running…") + "\n")
	}
	for i := range p.inputs {
		cur := "  "
		if i == p.fieldCur {
			cur = m.styl.selected.Render("› ")
		}
		b.WriteString(cur + m.styl.subtle.Render(fmt.Sprintf("%-14s", p.fieldKeys[i])) + " " + p.inputs[i].View() + "\n")
	}
	if p.errMsg != "" {
		b.WriteString("\n" + m.styl.errText.Render("✗ "+p.errMsg) + "\n")
	}
	return b.String()
}

func (m model) promptsResultView(p promptsState) string {
	var b strings.Builder
	b.WriteString(m.styl.label.Render("Rendered prompt") + "  " + m.styl.subtle.Render(fmt.Sprintf("(%d chars)", len(p.result))) + "\n\n")
	b.WriteString(p.result + "\n")
	if m.flash != "" {
		b.WriteString("\n" + m.styl.flash.Render("✓ "+m.flash) + "\n")
	}
	return b.String()
}

func (m model) promptsFooter() string {
	switch m.prompts.phase {
	case promptsFill:
		return "tab: next field · enter: next / run · esc: back to list"
	case promptsResult:
		return "y: copy to clipboard · esc: back to list"
	default:
		return "↑/↓: pick · enter: fill · r: reload · esc: dashboard · 1-6: switch · q: quit"
	}
}
