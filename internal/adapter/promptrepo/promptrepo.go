// Package promptrepo implements usecase/prompt.Repo by reading prompt templates
// from two fs.FS layers: the embedded defaults and the workspace's prompts/
// directory. An appdir copy shadows the embedded default of the same name,
// mirroring how templates are resolved.
package promptrepo

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

const subdir = "prompts"

// Repo reads prompts from embedded (required) and appdir (optional) FSes.
type Repo struct {
	Embedded fs.FS // the defaultAppDir skeleton, rooted so prompts/ is visible
	Appdir   fs.FS // os.DirFS(appdir); may be nil
}

// New returns a prompt.Repo backed by the embedded skeleton and, when non-nil,
// the workspace appdir.
func New(embedded, appdir fs.FS) prompt.Repo { return Repo{Embedded: embedded, Appdir: appdir} }

func (r Repo) List(ctx context.Context) ([]prompt.Entry, error) {
	names, overridden, err := r.names()
	if err != nil {
		return nil, err
	}
	entries := make([]prompt.Entry, 0, len(names))
	for _, name := range names {
		t, err := r.Load(ctx, name)
		if err != nil {
			// One malformed template must not hide every other prompt: surface
			// it as an unusable row rather than failing the whole listing.
			entries = append(entries, prompt.Entry{
				Name:        name,
				Description: "(unparseable: " + firstLine(err.Error()) + ")",
				Overridden:  overridden[name],
			})
			continue
		}
		entries = append(entries, prompt.Entry{
			Name:        t.Name,
			Description: t.Description,
			Overridden:  overridden[name],
		})
	}
	return entries, nil
}

func (r Repo) Load(_ context.Context, name string) (prompt.PromptTemplate, error) {
	path := subdir + "/" + name + ".md"

	if r.Appdir != nil {
		if raw, err := fs.ReadFile(r.Appdir, path); err == nil {
			return parseNamed(raw, name)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return prompt.PromptTemplate{}, fmt.Errorf("read %s: %w", path, err)
		}
	}

	raw, err := fs.ReadFile(r.Embedded, path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return prompt.PromptTemplate{}, fmt.Errorf("no prompt named %q", name)
		}
		return prompt.PromptTemplate{}, fmt.Errorf("read %s: %w", path, err)
	}
	return parseNamed(raw, name)
}

// parseNamed parses raw and enforces that its frontmatter name matches the file
// it was loaded from, so a listed name is always the name `run <name>` expects.
func parseNamed(raw []byte, name string) (prompt.PromptTemplate, error) {
	t, err := prompt.Parse(raw)
	if err != nil {
		return prompt.PromptTemplate{}, err
	}
	if t.Name != name {
		return prompt.PromptTemplate{}, fmt.Errorf("frontmatter name %q does not match file %q.md", t.Name, name)
	}
	return t, nil
}

// firstLine returns s up to the first newline, for compact list rows.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// names returns the sorted union of embedded and appdir prompt names, plus a
// set marking which names are shadowed by an appdir copy.
func (r Repo) names() ([]string, map[string]bool, error) {
	set := map[string]struct{}{}
	overridden := map[string]bool{}

	embNames, err := listNames(r.Embedded)
	if err != nil {
		return nil, nil, err
	}
	for _, n := range embNames {
		set[n] = struct{}{}
	}

	appNames, err := listNames(r.Appdir)
	if err != nil {
		return nil, nil, err
	}
	for _, n := range appNames {
		if _, ok := set[n]; ok {
			overridden[n] = true
		}
		set[n] = struct{}{}
	}

	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, overridden, nil
}

// listNames returns the base names (without .md) of prompt files in fsys's
// prompts/ dir. A nil fsys or a missing prompts/ dir yields no names.
func listNames(fsys fs.FS) ([]string, error) {
	if fsys == nil {
		return nil, nil
	}
	ents, err := fs.ReadDir(fsys, subdir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s dir: %w", subdir, err)
	}
	var names []string
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".md"))
	}
	return names, nil
}
