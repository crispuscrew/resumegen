//go:build !notui

package tui

import (
	"fmt"
	"strings"
)

func (model model) promptsView() string {
	state := model.prompts
	switch state.phase {
	case promptsFill:
		return model.promptsFillView(state)
	case promptsResult:
		return model.promptsResultView(state)
	default:
		return model.promptsPickView(state)
	}
}

func (model model) promptsPickView(state promptsState) string {
	if !state.loaded {
		return model.styl.subtle.Render("loading...")
	}
	if state.loadErr != nil {
		return model.styl.errText.Render("error: " + state.loadErr.Error())
	}
	if len(state.entries) == 0 {
		return model.styl.subtle.Render("no prompt templates found.")
	}
	var output strings.Builder
	output.WriteString(model.styl.label.Render("Prompt templates") + "\n\n")
	for index, entry := range state.entries {
		marker := ""
		if entry.Overridden {
			marker = model.styl.badge.Render(" (custom)")
		}
		if index == state.cursor {
			output.WriteString(model.styl.selected.Render("> "+entry.Name) + marker + "\n")
			if entry.Description != "" {
				output.WriteString("    " + model.styl.subtle.Render(entry.Description) + "\n")
			}
		} else {
			output.WriteString("  " + entry.Name + marker + "\n")
		}
	}
	if state.errMsg != "" {
		output.WriteString("\n" + model.styl.errText.Render("error: "+state.errMsg) + "\n")
	}
	return output.String()
}

func (model model) promptsFillView(state promptsState) string {
	var output strings.Builder
	output.WriteString(model.styl.title.Render(state.form.Name) + "\n")
	if state.form.Description != "" {
		output.WriteString(model.styl.subtle.Render(state.form.Description) + "\n")
	}
	output.WriteString("\n")
	if len(state.inputs) == 0 {
		output.WriteString(model.styl.subtle.Render("no inputs needed - running...") + "\n")
	}
	for index := range state.inputs {
		cursor := "  "
		if index == state.fieldCur {
			cursor = model.styl.selected.Render("> ")
		}
		output.WriteString(cursor + model.styl.subtle.Render(fmt.Sprintf("%-14s", state.fieldKeys[index])) + " " + state.inputs[index].View() + "\n")
	}
	if state.errMsg != "" {
		output.WriteString("\n" + model.styl.errText.Render("error: "+state.errMsg) + "\n")
	}
	return output.String()
}

func (model model) promptsResultView(state promptsState) string {
	var output strings.Builder
	output.WriteString(model.styl.label.Render("Rendered prompt") + "  " + model.styl.subtle.Render(fmt.Sprintf("(%d chars)", len(state.result))) + "\n\n")
	output.WriteString(state.result + "\n")
	if model.flash != "" {
		output.WriteString("\n" + model.styl.flash.Render("done: "+model.flash) + "\n")
	}
	return output.String()
}

func (model model) promptsFooter() string {
	switch model.prompts.phase {
	case promptsFill:
		return "tab: next field | enter: next / run | esc: back to list"
	case promptsResult:
		return "y: copy to clipboard | esc: back to list"
	default:
		return "up/down: pick | enter: fill | r: reload | esc: dashboard | 1-6: switch | q: quit"
	}
}
