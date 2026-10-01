package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/crispuscrew/resumegen/internal/adapter/clipboard"
	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

func inputSources(template prompt.PromptTemplate) map[string]string {
	sources := make(map[string]string, len(template.Inputs))
	for key, spec := range template.Inputs {
		sources[key] = spec.Source
	}
	return sources
}

func emitPrompt(ctx context.Context, template prompt.PromptTemplate, text, output string, copyOut, jsonOut bool) error {
	result := runJSON{Prompt: template.Name, Text: text, Inputs: inputSources(template), Chars: len(text)}
	switch {
	case copyOut:
		if err := clipboard.Copy(ctx, text); err != nil {
			return err
		}
		result.Copied = true
	case output != "":
		if err := os.WriteFile(output, []byte(text), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", output, err)
		}
		result.Output = output
	}
	if jsonOut {
		return emitJSON(result)
	}
	switch {
	case copyOut:
		fmt.Fprintf(os.Stderr, "copied to clipboard (%d chars)\n", len(text))
	case output != "":
		fmt.Fprintf(os.Stderr, "wrote %s (%d chars)\n", output, len(text))
	default:
		fmt.Print(text)
	}
	return nil
}
