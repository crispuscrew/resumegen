package promptrepo_test

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/crispuscrew/resumegen"
	"github.com/crispuscrew/resumegen/internal/adapter/promptrepo"
)

func embeddedSkeleton(t *testing.T) fs.FS {
	t.Helper()
	sk, err := fs.Sub(resumegen.Defaults, "defaultAppDir")
	if err != nil {
		t.Fatal(err)
	}
	return sk
}

// Every shipped prompt must parse — this guards the frontmatter/placeholder
// symmetry of the whole embedded corpus.
func TestEmbeddedPrompts_AllParse(t *testing.T) {
	repo := promptrepo.New(embeddedSkeleton(t), nil)
	entries, err := repo.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 9 {
		t.Fatalf("want at least 9 embedded prompts, got %d", len(entries))
	}
	for _, e := range entries {
		if _, err := repo.Load(context.Background(), e.Name); err != nil {
			t.Errorf("embedded prompt %q failed to load: %v", e.Name, err)
		}
		if e.Overridden {
			t.Errorf("embedded-only prompt %q reported as overridden", e.Name)
		}
	}
}

func TestRepo_AppdirShadowsEmbedded(t *testing.T) {
	overlay := fstest.MapFS{
		"prompts/tailor-bullets.md": &fstest.MapFile{Data: []byte(`+++
name        = "tailor-bullets"
description = "MY custom version"
+++
custom body`)},
	}
	repo := promptrepo.New(embeddedSkeleton(t), overlay)

	tpl, err := repo.Load(context.Background(), "tailor-bullets")
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Description != "MY custom version" {
		t.Errorf("appdir copy should win, got %q", tpl.Description)
	}

	entries, err := repo.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, e := range entries {
		if e.Name == "tailor-bullets" {
			found = true
			if !e.Overridden {
				t.Errorf("shadowed prompt should be marked Overridden")
			}
		}
	}
	if !found {
		t.Error("tailor-bullets missing from listing")
	}
}

func TestRepo_UnknownPrompt(t *testing.T) {
	repo := promptrepo.New(embeddedSkeleton(t), nil)
	if _, err := repo.Load(context.Background(), "does-not-exist"); err == nil {
		t.Error("expected error for unknown prompt")
	}
}

// Load must reject a template whose frontmatter name disagrees with its
// filename, so a listed name is always runnable.
func TestRepo_NameMustMatchFilename(t *testing.T) {
	overlay := fstest.MapFS{
		"prompts/mislabeled.md": &fstest.MapFile{Data: []byte(`+++
name        = "something-else"
description = "d"
+++
body`)},
	}
	repo := promptrepo.New(embeddedSkeleton(t), overlay)
	if _, err := repo.Load(context.Background(), "mislabeled"); err == nil {
		t.Error("expected name/filename mismatch to error")
	}
}

// A single malformed template must not hide the rest of the listing.
func TestRepo_ListTolerantOfBadTemplate(t *testing.T) {
	overlay := fstest.MapFS{
		"prompts/broken.md": &fstest.MapFile{Data: []byte("no frontmatter here")},
	}
	repo := promptrepo.New(embeddedSkeleton(t), overlay)
	entries, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("list should tolerate a bad template, got %v", err)
	}
	var broken bool
	for _, e := range entries {
		if e.Name == "broken" {
			broken = true
			if !strings.Contains(e.Description, "unparseable") {
				t.Errorf("bad template should be flagged unparseable, got %q", e.Description)
			}
		}
	}
	if !broken {
		t.Error("broken template missing from listing")
	}
	if len(entries) < 10 {
		t.Errorf("other prompts should still list, got %d", len(entries))
	}
}
