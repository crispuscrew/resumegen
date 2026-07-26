package tomlrepo

import (
	"testing"
	"testing/fstest"
)

func TestListProfiles(t *testing.T) {
	fsys := fstest.MapFS{
		"profiles/default.toml":      {Data: []byte("")},
		"profiles/cpp-embedded.toml": {Data: []byte("")},
		"profiles/notes.md":          {Data: []byte("")}, // ignored: not .toml
		"profiles/sub/x.toml":        {Data: []byte("")}, // ignored: nested dir entry
		"profiles/.#default.toml":    {Data: []byte("")}, // ignored: editor lock dotfile
		"profiles/.toml":             {Data: []byte("")}, // ignored: would be an empty name
		"data/header.toml":           {Data: []byte("")}, // ignored: wrong dir
	}
	got, err := ListProfiles(fsys)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"cpp-embedded", "default"} // sorted, .toml only, top-level only
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestListProfiles_NoDir(t *testing.T) {
	got, err := ListProfiles(fstest.MapFS{"config.toml": {Data: []byte("")}})
	if err != nil {
		t.Fatalf("missing profiles/ should not error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
}
