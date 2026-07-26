package prompt

import "context"

// Entry is one row of `prompt list`: identity plus whether an appdir copy
// shadows the embedded default.
type Entry struct {
	Name        string
	Description string
	Overridden  bool
}

// Repo loads prompt templates from the embedded defaults unioned with the
// workspace's prompts/ directory (appdir shadows embedded, same rule as
// templates). Implementations live in the adapter layer.
type Repo interface {
	// List returns every available template, sorted by name.
	List(ctx context.Context) ([]Entry, error)
	// Load parses and returns the template named name, preferring an appdir
	// copy over the embedded default.
	Load(ctx context.Context, name string) (PromptTemplate, error)
}
