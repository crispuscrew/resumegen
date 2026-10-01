//go:build !notui

package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Context sources get a shared profile/application field, not a text input.
func buildPromptInputs(form PromptForm) ([]textinput.Model, []string) {
	var inputs []textinput.Model
	var keys []string
	needProfile, needApp := false, false
	add := func(key, placeholder, defaultValue string) {
		input := newTextInput(placeholder)
		if defaultValue != "" {
			input.SetValue(defaultValue)
		}
		inputs = append(inputs, input)
		keys = append(keys, key)
	}
	for _, field := range form.Fields {
		switch field.Source {
		case "flag", "prompt", "stdin":
			add(field.Key, fieldPlaceholder(field), field.Default)
		case "file", "jd-file":
			// A file default is fallback content, not a default path.
			add(field.Key, "path to a "+field.Key+" file", "")
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
		for index := 1; index < len(inputs); index++ {
			inputs[index].Blur()
		}
	}
	return inputs, keys
}

func fieldPlaceholder(field PromptField) string {
	if field.Required {
		return field.Key + " (required)"
	}
	return field.Key
}

func (model model) loadPrompts() tea.Cmd {
	ctx, list := model.deps.Ctx, model.deps.ListPrompts
	return func() tea.Msg {
		if list == nil {
			return promptsLoadedMsg{err: errNoCapability}
		}
		entries, err := list(ctx)
		return promptsLoadedMsg{entries: entries, err: err}
	}
}

func (model model) loadPromptForm(name string) tea.Cmd {
	ctx, load := model.deps.Ctx, model.deps.LoadPrompt
	return func() tea.Msg {
		if load == nil {
			return promptFormMsg{err: errNoCapability}
		}
		form, err := load(ctx, name)
		return promptFormMsg{form: form, err: err}
	}
}

func (model model) runPromptCmd() tea.Cmd {
	state := model.prompts
	values := make(map[string]string, len(state.fieldKeys))
	for index, key := range state.fieldKeys {
		values[key] = state.inputs[index].Value()
	}
	name := state.form.Name
	ctx, run := model.deps.Ctx, model.deps.RunPrompt
	return func() tea.Msg {
		if run == nil {
			return promptRunMsg{err: errNoCapability}
		}
		text, err := run(ctx, name, values)
		return promptRunMsg{text: text, err: err}
	}
}

func (model model) copyCmd(text, label string) tea.Cmd {
	ctx, copyText := model.deps.Ctx, model.deps.Copy
	return func() tea.Msg {
		if copyText == nil {
			return copyDoneMsg{err: errNoCapability}
		}
		return copyDoneMsg{err: copyText(ctx, text), label: label}
	}
}
