//go:build !notui

package tui

import "testing"

func TestBuildPromptInputs_FileRequiresPath(test *testing.T) {
	inputs, keys := buildPromptInputs(PromptForm{Fields: []PromptField{
		{Key: "resume", Source: "file", Required: true, Default: "fallback content"},
	}})
	if len(inputs) != 1 || len(keys) != 1 || keys[0] != "resume" {
		test.Fatalf("expected one resume file input, got %v", keys)
	}
	if inputs[0].Value() != "" || inputs[0].Placeholder != "path to a resume file" {
		test.Fatalf("file input must ask for a path, got %+v", inputs[0])
	}
}
