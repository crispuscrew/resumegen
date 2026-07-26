package cli

import (
	"encoding/json"
	"testing"

	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

func TestListJSON_Shape(t *testing.T) {
	got := listJSON([]prompt.Entry{{Name: "a", Description: "d", Overridden: true}})
	b, _ := json.Marshal(got)
	var back []map[string]any
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	row := back[0]
	for _, key := range []string{"name", "description", "overridden"} {
		if _, ok := row[key]; !ok {
			t.Errorf("list JSON missing key %q: %v", key, row)
		}
	}
}

func TestTemplateJSON_Shape(t *testing.T) {
	tpl := prompt.PromptTemplate{
		Name:        "x",
		Description: "d",
		Body:        "hi {{who}}",
		Inputs: map[string]prompt.InputSpec{
			"who": {Source: prompt.SourceFlag, Flag: "who", Required: true},
		},
	}
	b, _ := json.Marshal(templateJSON(tpl))
	var obj map[string]any
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"name", "description", "inputs", "body"} {
		if _, ok := obj[key]; !ok {
			t.Errorf("show JSON missing key %q", key)
		}
	}
	inputs := obj["inputs"].(map[string]any)
	who := inputs["who"].(map[string]any)
	if who["source"] != "flag" || who["required"] != true {
		t.Errorf("input spec JSON wrong: %v", who)
	}
}

func TestRunJSON_Shape(t *testing.T) {
	obj := runJSON{Prompt: "p", Text: "t", Inputs: map[string]string{"jd": "jd-file"}, Chars: 1, Copied: true}
	b, _ := json.Marshal(obj)
	var back map[string]any
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"prompt", "text", "inputs", "chars", "copied"} {
		if _, ok := back[key]; !ok {
			t.Errorf("run JSON missing key %q", key)
		}
	}
	// output omitted when empty
	if _, ok := back["output"]; ok {
		t.Errorf("output should be omitted when unset")
	}
}
