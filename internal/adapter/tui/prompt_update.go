//go:build !notui

package tui

import tea "github.com/charmbracelet/bubbletea"

func (model model) updatePrompts(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch model.prompts.phase {
	case promptsPickList:
		return model.updatePromptsPick(message)
	case promptsFill:
		return model.updatePromptsFill(message)
	case promptsResult:
		return model.updatePromptsResult(message)
	}
	return model, nil
}

func (model model) updatePromptsPick(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	state := &model.prompts
	switch message.String() {
	case "esc":
		model = model.switchTo(screenDashboard)
		return model, nil
	case "up", "k":
		if state.cursor > 0 {
			state.cursor--
		}
		return model, nil
	case "down", "j":
		if state.cursor < len(state.entries)-1 {
			state.cursor++
		}
		return model, nil
	case "r":
		state.loaded = false
		return model, model.loadPrompts()
	case "enter":
		if len(state.entries) == 0 {
			return model, nil
		}
		return model, model.loadPromptForm(state.entries[state.cursor].Name)
	}
	return model, nil
}

func (model model) updatePromptsFill(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	state := &model.prompts
	switch message.String() {
	case "esc":
		state.phase = promptsPickList
		state.inputs, state.fieldKeys, state.errMsg = nil, nil, ""
		return model, nil
	case "tab", "down":
		state.focusField(1)
		return model, nil
	case "shift+tab", "up":
		state.focusField(-1)
		return model, nil
	case "enter":
		if len(state.inputs) == 0 {
			return model, nil
		}
		if state.fieldCur < len(state.inputs)-1 {
			state.focusField(1)
			return model, nil
		}
		return model, model.runPromptCmd()
	}
	if len(state.inputs) > 0 {
		var command tea.Cmd
		state.inputs[state.fieldCur], command = state.inputs[state.fieldCur].Update(message)
		return model, command
	}
	return model, nil
}

func (model model) updatePromptsResult(message tea.KeyMsg) (tea.Model, tea.Cmd) {
	state := &model.prompts
	switch message.String() {
	case "esc":
		state.phase = promptsPickList
		state.result = ""
		return model, nil
	case "y":
		return model, model.copyCmd(state.result, "copied to clipboard")
	}
	return model, nil
}
