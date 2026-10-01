// Package prompt is the prompt-template layer: it parses TOML-frontmatter +
// Markdown templates, validates that their {{placeholder}} slots line up with
// the inputs they declare, and substitutes resolved values to produce a
// ready-to-paste LLM prompt. It performs no IO and calls no LLM - resolution
// of input sources and delivery to sinks live in the adapter/CLI layer.
package prompt

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// InputSpec declares a placeholder's source, CLI flag, default and requirements.
type InputSpec struct {
	Source   string `toml:"source"`
	Flag     string `toml:"flag"`
	Default  string `toml:"default"`
	Required bool   `toml:"required"`
	// Field selects which stored field an app-id input resolves to. Required
	// (and only meaningful) when Source == SourceAppID.
	Field string `toml:"field"`
}

const (
	SourceDataDump = "data-dump" // contents of output/<profile>.md (v1.2)
	SourceFile     = "file"      // contents of the file named by --<Flag>
	SourceJDFile   = "jd-file"   // file contents, with a tracked JD fallback
	SourceFlag     = "flag"      // raw --<Flag> value
	SourcePrompt   = "prompt"    // one line read interactively
	SourceStdin    = "stdin"     // all of stdin
	SourceAppID    = "app-id"    // a Field of the application named by --app (v1.4 tracker)
)

// "jd" resolves to the contents of the application's jd_path.
var appIDFields = map[string]struct{}{
	"company": {}, "role": {}, "status": {}, "source": {}, "notes": {}, "jd": {},
}

func validAppField(f string) bool { _, ok := appIDFields[f]; return ok }

// PromptTemplate is a parsed template: metadata + declared inputs + the raw
// Markdown body containing {{placeholder}} slots.
type PromptTemplate struct {
	Name        string               `toml:"name"`
	Description string               `toml:"description"`
	Inputs      map[string]InputSpec `toml:"inputs"`
	Body        string               `toml:"-"`
}

// PromptInput holds the resolved value for each placeholder key.
type PromptInput map[string]string

var (
	placeholderRe = regexp.MustCompile(`\{\{\s*([a-zA-Z][\w-]*)\s*\}\}`)
	frontmatterRe = regexp.MustCompile(`(?s)\A\+\+\+\r?\n(.*?)\r?\n\+\+\+\r?\n?(.*)\z`)
)

// Parse decodes frontmatter and checks input/placeholder symmetry.
func Parse(raw []byte) (PromptTemplate, error) {
	m := frontmatterRe.FindSubmatch(raw)
	if m == nil {
		return PromptTemplate{}, fmt.Errorf("template has no +++ TOML frontmatter")
	}

	var t PromptTemplate
	if err := toml.Unmarshal(m[1], &t); err != nil {
		return PromptTemplate{}, fmt.Errorf("parse frontmatter: %w", err)
	}
	t.Body = string(m[2])
	if t.Name == "" {
		return PromptTemplate{}, fmt.Errorf("template frontmatter is missing name")
	}

	for key, spec := range t.Inputs {
		if !validSource(spec.Source) {
			return PromptTemplate{}, fmt.Errorf("input %q has unknown source %q", key, spec.Source)
		}
		if spec.Source == SourceAppID && !validAppField(spec.Field) {
			return PromptTemplate{}, fmt.Errorf("input %q: source %q requires field to be one of {company,role,status,source,notes,jd}, got %q", key, SourceAppID, spec.Field)
		}
	}
	if err := checkSymmetry(t); err != nil {
		return PromptTemplate{}, err
	}
	return t, nil
}

// Render substitutes declared placeholders, rejecting missing required inputs.
func Render(t PromptTemplate, in PromptInput) (string, error) {
	var missing []string
	for key, spec := range t.Inputs {
		if _, ok := in[key]; !ok && spec.Required {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return "", fmt.Errorf("missing required input(s): %s", strings.Join(missing, ", "))
	}

	out := placeholderRe.ReplaceAllStringFunc(t.Body, func(match string) string {
		key := placeholderRe.FindStringSubmatch(match)[1]
		return in[key]
	})
	return out, nil
}

// Placeholders returns the sorted, de-duplicated set of {{keys}} in body.
func Placeholders(body string) []string {
	seen := map[string]struct{}{}
	for _, m := range placeholderRe.FindAllStringSubmatch(body, -1) {
		seen[m[1]] = struct{}{}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func checkSymmetry(t PromptTemplate) error {
	used := map[string]struct{}{}
	for _, key := range Placeholders(t.Body) {
		used[key] = struct{}{}
		if _, ok := t.Inputs[key]; !ok {
			return fmt.Errorf("placeholder {{%s}} has no matching [inputs.%s] table", key, key)
		}
	}
	for key := range t.Inputs {
		if _, ok := used[key]; !ok {
			return fmt.Errorf("input %q is declared but never used in the body", key)
		}
	}
	return nil
}

func validSource(s string) bool {
	switch s {
	case SourceDataDump, SourceFile, SourceJDFile, SourceFlag, SourcePrompt, SourceStdin, SourceAppID:
		return true
	default:
		return false
	}
}
