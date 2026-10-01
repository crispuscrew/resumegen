//go:build !notui

package tui

import (
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
	inputs    []textinput.Model
	fieldKeys []string
	fieldCur  int
	result    string
	errMsg    string
}

func (state promptsState) capturing() bool {
	return state.phase == promptsFill && len(state.inputs) > 0
}

func (state *promptsState) focusField(delta int) {
	if len(state.inputs) == 0 {
		return
	}
	state.inputs[state.fieldCur].Blur()
	state.fieldCur = (state.fieldCur + delta + len(state.inputs)) % len(state.inputs)
	state.inputs[state.fieldCur].Focus()
}

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
	label string
}

func (model model) onPromptsLoaded(message promptsLoadedMsg) (tea.Model, tea.Cmd) {
	model.prompts.loaded = true
	model.prompts.entries = message.entries
	model.prompts.loadErr = message.err
	if model.prompts.cursor >= len(model.prompts.entries) {
		model.prompts.cursor = maxInt(0, len(model.prompts.entries)-1)
	}
	return model, nil
}

func (model model) onPromptForm(message promptFormMsg) (tea.Model, tea.Cmd) {
	state := &model.prompts
	// Ignore a form delivered after navigation or another form was opened.
	if model.active != screenPrompts || state.phase != promptsPickList {
		return model, nil
	}
	if message.err != nil {
		state.errMsg = message.err.Error()
		return model, nil
	}
	state.form = message.form
	state.inputs, state.fieldKeys = buildPromptInputs(message.form)
	state.fieldCur = 0
	state.errMsg = ""
	state.phase = promptsFill
	if len(state.inputs) == 0 {
		return model, model.runPromptCmd()
	}
	return model, nil
}

func (model model) onPromptRun(message promptRunMsg) (tea.Model, tea.Cmd) {
	state := &model.prompts
	// A result belongs only to the fill form that requested it.
	if model.active != screenPrompts || state.phase != promptsFill {
		return model, nil
	}
	if message.err != nil {
		state.errMsg = message.err.Error()
		return model, nil
	}
	state.result = message.text
	model.flash = ""
	state.phase = promptsResult
	return model, nil
}

func (model model) onCopyDone(message copyDoneMsg) (tea.Model, tea.Cmd) {
	switch {
	case message.err != nil:
		model.flash = "copy failed: " + message.err.Error()
	case message.label != "":
		model.flash = message.label
	default:
		model.flash = "copied to clipboard"
	}
	return model, nil
}
