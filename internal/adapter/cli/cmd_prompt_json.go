package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

// Stable JSON shapes for the agent-facing contract (--json). These DTOs pin the
// wire format independently of the internal types, so refactors can't silently
// change the contract.

type listItemJSON struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Overridden  bool   `json:"overridden"`
}

type inputJSON struct {
	Source   string `json:"source"`
	Flag     string `json:"flag,omitempty"`
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required"`
	Field    string `json:"field,omitempty"`
}

type showJSON struct {
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Inputs      map[string]inputJSON `json:"inputs"`
	Body        string               `json:"body"`
}

type runJSON struct {
	Prompt string            `json:"prompt"`
	Text   string            `json:"text"`
	Inputs map[string]string `json:"inputs"`
	Chars  int               `json:"chars"`
	Copied bool              `json:"copied,omitempty"`
	Output string            `json:"output,omitempty"`
}

func listJSON(entries []prompt.Entry) []listItemJSON {
	out := make([]listItemJSON, 0, len(entries))
	for _, e := range entries {
		out = append(out, listItemJSON{Name: e.Name, Description: e.Description, Overridden: e.Overridden})
	}
	return out
}

func templateJSON(t prompt.PromptTemplate) showJSON {
	inputs := make(map[string]inputJSON, len(t.Inputs))
	for k, s := range t.Inputs {
		inputs[k] = inputJSON{Source: s.Source, Flag: s.Flag, Default: s.Default, Required: s.Required, Field: s.Field}
	}
	return showJSON{Name: t.Name, Description: t.Description, Inputs: inputs, Body: t.Body}
}

// emitJSON writes v as indented JSON to stdout with a trailing newline.
func emitJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	writeln(os.Stdout, string(b))
	return nil
}
