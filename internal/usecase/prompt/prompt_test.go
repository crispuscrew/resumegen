package prompt_test

import (
	"strings"
	"testing"

	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

const valid = `+++
name        = "greet"
description = "say hi"

[inputs.who]
source   = "flag"
flag     = "who"
required = true
+++
Hello {{who}}!`

func TestParse_AppIDField(t *testing.T) {
	tmpl := func(field string) []byte {
		f := ""
		if field != "" {
			f = "field  = \"" + field + "\"\n"
		}
		return []byte("+++\nname = \"x\"\n[inputs.jd]\nsource = \"app-id\"\n" + f + "+++\n{{jd}}")
	}
	if _, err := prompt.Parse(tmpl("jd")); err != nil {
		t.Fatalf("app-id with field=jd should parse: %v", err)
	}
	if _, err := prompt.Parse(tmpl("")); err == nil {
		t.Error("app-id without a field should error")
	}
	if _, err := prompt.Parse(tmpl("salary")); err == nil {
		t.Error("app-id with an unknown field should error")
	}
}

func TestParse_Valid(t *testing.T) {
	tpl, err := prompt.Parse([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Name != "greet" || tpl.Description != "say hi" {
		t.Errorf("metadata wrong: %+v", tpl)
	}
	if tpl.Body != "Hello {{who}}!" {
		t.Errorf("body = %q", tpl.Body)
	}
	spec, ok := tpl.Inputs["who"]
	if !ok || spec.Source != prompt.SourceFlag || spec.Flag != "who" || !spec.Required {
		t.Errorf("input spec wrong: %+v", tpl.Inputs)
	}
}

func TestParse_Errors(t *testing.T) {
	cases := map[string]string{
		"no frontmatter": "Hello {{who}}",
		"missing name": `+++
description = "x"
+++
body`,
		"placeholder without input": `+++
name = "x"
+++
Hello {{who}}`,
		"input without placeholder": `+++
name = "x"
[inputs.who]
source = "flag"
+++
Hello`,
		"unknown source": `+++
name = "x"
[inputs.who]
source = "carrier-pigeon"
+++
Hello {{who}}`,
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := prompt.Parse([]byte(raw)); err == nil {
				t.Errorf("expected parse error for %q", name)
			}
		})
	}
}

func TestRender_Substitutes(t *testing.T) {
	tpl, err := prompt.Parse([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	out, err := prompt.Render(tpl, prompt.PromptInput{"who": "world"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello world!" {
		t.Errorf("got %q", out)
	}
}

func TestRender_MissingRequired(t *testing.T) {
	tpl, _ := prompt.Parse([]byte(valid))
	_, err := prompt.Render(tpl, prompt.PromptInput{})
	if err == nil || !strings.Contains(err.Error(), "who") {
		t.Errorf("want error naming 'who', got %v", err)
	}
}

func TestRender_OptionalMissingBecomesEmpty(t *testing.T) {
	raw := `+++
name = "x"
[inputs.who]
source = "flag"
required = false
+++
Hi{{who}}`
	tpl, err := prompt.Parse([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	out, err := prompt.Render(tpl, prompt.PromptInput{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hi" {
		t.Errorf("optional-missing should substitute empty, got %q", out)
	}
}

func TestRender_RepeatedPlaceholder(t *testing.T) {
	raw := `+++
name = "x"
[inputs.a]
source = "flag"
required = true
+++
{{a}}-{{a}}`
	tpl, _ := prompt.Parse([]byte(raw))
	out, _ := prompt.Render(tpl, prompt.PromptInput{"a": "z"})
	if out != "z-z" {
		t.Errorf("got %q", out)
	}
}

func TestPlaceholders_DedupSorted(t *testing.T) {
	got := prompt.Placeholders("{{b}} {{a}} {{b}} {{ a }}")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("got %v, want [a b]", got)
	}
}
