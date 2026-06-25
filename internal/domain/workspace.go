package domain

// CurrentWorkspaceSchema is the schema version written by `resumegen init`
// for newly-created workspace markers. Loaders accept any value as long as
// the marker is well-formed TOML.
const CurrentWorkspaceSchema = "1.1"

// Workspace is the parsed `.resumegen/workspace.toml` marker. Presence of the
// marker file (not the contents) is what makes a directory a workspace; the
// fields here are metadata for tooling and humans.
type Workspace struct {
	SchemaVersion string        `toml:"schema_version"`
	Workspace     WorkspaceMeta `toml:"workspace"`
}

// WorkspaceMeta holds human-facing identification for a workspace. Both
// fields are optional; init defaults Name to the workspace directory's
// basename.
type WorkspaceMeta struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
}
