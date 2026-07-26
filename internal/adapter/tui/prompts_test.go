//go:build !notui

package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBuildPromptInputs(t *testing.T) {
	form := PromptForm{
		Name: "x",
		Fields: []PromptField{
			{Key: "role", Source: "flag", Required: true},
			{Key: "resume", Source: "data-dump"},
			{Key: "jd", Source: "jd-file"},
			{Key: "co", Source: "app-id", Field: "company"},
			{Key: "jdtext", Source: "app-id", Field: "jd"},
		},
	}
	inputs, keys := buildPromptInputs(form)
	// flag(role) + jd-file(jd) + one __profile (data-dump) + one __app (both app-id)
	want := []string{"role", "jd", "__profile", "__app"}
	if len(keys) != len(want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys = %v, want %v", keys, want)
		}
	}
	if len(inputs) != 4 {
		t.Fatalf("inputs = %d, want 4", len(inputs))
	}
	// __profile pre-fills to "default"
	if inputs[2].Value() != "default" {
		t.Fatalf("__profile default = %q, want default", inputs[2].Value())
	}
}

func TestPrompts_FillRunCopy(t *testing.T) {
	m := newTestModel(newMemStore())
	m.deps.ListPrompts = func(context.Context) ([]PromptEntry, error) {
		return []PromptEntry{{Name: "greet", Description: "say hi"}}, nil
	}
	m.deps.LoadPrompt = func(_ context.Context, name string) (PromptForm, error) {
		return PromptForm{Name: name, Fields: []PromptField{{Key: "who", Source: "flag", Required: true}}}, nil
	}
	var gotValues map[string]string
	m.deps.RunPrompt = func(_ context.Context, _ string, values map[string]string) (string, error) {
		gotValues = values
		return "Hello " + values["who"], nil
	}

	// enter prompts, pick the first template
	m2, cmd := step(m, runeKey("4"))
	m = drain(m2, cmd)
	if m.active != screenPrompts || m.prompts.phase != promptsPickList {
		t.Fatalf("not on prompt picker: active=%d phase=%d", m.active, m.prompts.phase)
	}
	m = press(m, namedKey(tea.KeyEnter)) // load form
	if m.prompts.phase != promptsFill {
		t.Fatalf("phase should be fill, got %d", m.prompts.phase)
	}
	// type the value and run
	m = press(m, runeKey("world"))
	m = press(m, namedKey(tea.KeyEnter))
	if m.prompts.phase != promptsResult {
		t.Fatalf("phase should be result, got %d", m.prompts.phase)
	}
	if m.prompts.result != "Hello world" {
		t.Fatalf("result = %q", m.prompts.result)
	}
	if gotValues["who"] != "world" {
		t.Fatalf("RunPrompt got values %v", gotValues)
	}

	// copy the result
	var copied string
	m.deps.Copy = func(_ context.Context, text string) error { copied = text; return nil }
	m = press(m, runeKey("y"))
	if copied != "Hello world" {
		t.Fatalf("copied = %q", copied)
	}
}

func TestPrompts_EscFromFillReturnsToPicker(t *testing.T) {
	m := newTestModel(newMemStore())
	m.deps.ListPrompts = func(context.Context) ([]PromptEntry, error) {
		return []PromptEntry{{Name: "greet"}}, nil
	}
	m.deps.LoadPrompt = func(_ context.Context, name string) (PromptForm, error) {
		return PromptForm{Name: name, Fields: []PromptField{{Key: "who", Source: "flag"}}}, nil
	}
	m2, cmd := step(m, runeKey("4"))
	m = drain(m2, cmd)
	m = press(m, namedKey(tea.KeyEnter)) // fill
	m = press(m, namedKey(tea.KeyEsc))   // cancel
	if m.prompts.phase != promptsPickList {
		t.Fatalf("esc should return to picker, phase=%d", m.prompts.phase)
	}
	if m.prompts.inputs != nil {
		t.Fatal("inputs should be cleared on cancel")
	}
}

// TestPrompts_StaleRunDroppedAfterEsc: a run result arriving after the user
// esc'd back to the picker must be dropped, not force them into a result view.
func TestPrompts_StaleRunDroppedAfterEsc(t *testing.T) {
	m := newTestModel(newMemStore())
	m.deps.ListPrompts = func(context.Context) ([]PromptEntry, error) {
		return []PromptEntry{{Name: "greet"}}, nil
	}
	m.deps.LoadPrompt = func(_ context.Context, name string) (PromptForm, error) {
		return PromptForm{Name: name, Fields: []PromptField{{Key: "who", Source: "flag"}}}, nil
	}
	m.deps.RunPrompt = func(_ context.Context, _ string, _ map[string]string) (string, error) {
		return "late result", nil
	}
	m2, cmd := step(m, runeKey("4"))
	m = drain(m2, cmd)
	m = press(m, namedKey(tea.KeyEnter)) // fill
	m = press(m, runeKey("x"))
	// dispatch the run but DON'T deliver its msg yet
	m2, runCmd := step(m, namedKey(tea.KeyEnter))
	m = m2
	m = press(m, namedKey(tea.KeyEsc)) // back to picker before the result lands
	if m.prompts.phase != promptsPickList {
		t.Fatalf("precondition: expected picker, phase=%d", m.prompts.phase)
	}
	m = drain(m, runCmd) // late delivery
	if m.prompts.phase != promptsPickList {
		t.Fatalf("stale run result must be dropped, phase=%d", m.prompts.phase)
	}
	if m.prompts.result != "" {
		t.Fatalf("stale result stored: %q", m.prompts.result)
	}
}

// TestPrompts_SecondFormMsgIgnored: double-enter on the picker dispatches two
// form loads; the second arriving must not wipe the form the user is typing in.
func TestPrompts_SecondFormMsgIgnored(t *testing.T) {
	m := newTestModel(newMemStore())
	m.deps.ListPrompts = func(context.Context) ([]PromptEntry, error) {
		return []PromptEntry{{Name: "greet"}}, nil
	}
	m.deps.LoadPrompt = func(_ context.Context, name string) (PromptForm, error) {
		return PromptForm{Name: name, Fields: []PromptField{{Key: "who", Source: "flag"}}}, nil
	}
	m2, cmd := step(m, runeKey("4"))
	m = drain(m2, cmd)
	// two enters on the picker: capture both load commands
	m2, load1 := step(m, namedKey(tea.KeyEnter))
	m2b, load2 := step(m2, namedKey(tea.KeyEnter))
	m = drain(m2b, load1) // first form arrives -> fill
	if m.prompts.phase != promptsFill {
		t.Fatalf("expected fill, phase=%d", m.prompts.phase)
	}
	m = press(m, runeKey("typed")) // user types
	m = drain(m, load2)            // second (stale) form arrives
	if got := m.prompts.inputs[0].Value(); got != "typed" {
		t.Fatalf("stale form msg wiped the user's input: %q", got)
	}
}

// TestPrompts_ZeroInputEnterDoesNotRerun: with a zero-input form the auto-run
// is already in flight; enter must not dispatch a second run.
func TestPrompts_ZeroInputEnterDoesNotRerun(t *testing.T) {
	runs := 0
	m := newTestModel(newMemStore())
	m.deps.ListPrompts = func(context.Context) ([]PromptEntry, error) {
		return []PromptEntry{{Name: "auto"}}, nil
	}
	m.deps.LoadPrompt = func(_ context.Context, name string) (PromptForm, error) {
		return PromptForm{Name: name, Fields: []PromptField{{Key: "resume", Source: "data-dump"}}}, nil
	}
	m.deps.RunPrompt = func(_ context.Context, _ string, _ map[string]string) (string, error) {
		runs++
		return "out", nil
	}
	// buildPromptInputs adds __profile for data-dump, so force a truly
	// zero-input form: use a template with no collected fields at all.
	m.deps.LoadPrompt = func(_ context.Context, name string) (PromptForm, error) {
		return PromptForm{Name: name}, nil
	}
	m2, cmd := step(m, runeKey("4"))
	m = drain(m2, cmd)
	// enter -> form msg -> auto-run dispatch; capture the run cmd undelivered
	m2, formCmd := step(m, namedKey(tea.KeyEnter))
	m, runCmd := step(m2, formCmd())
	if m.prompts.phase != promptsFill || len(m.prompts.inputs) != 0 {
		t.Fatalf("expected zero-input fill, phase=%d inputs=%d", m.prompts.phase, len(m.prompts.inputs))
	}
	// enter while the auto-run is in flight: must not dispatch another
	var extra tea.Cmd
	m, extra = step(m, namedKey(tea.KeyEnter))
	if extra != nil {
		t.Fatal("enter re-dispatched the auto-run")
	}
	m = drain(m, runCmd)
	if runs != 1 {
		t.Fatalf("want exactly 1 run, got %d", runs)
	}
	if m.prompts.phase != promptsResult {
		t.Fatalf("auto-run result should land, phase=%d", m.prompts.phase)
	}
}

func TestConfigAndDataLoad(t *testing.T) {
	base := newTestModel(newMemStore())
	base.deps.LoadConfig = func(context.Context) (ConfigView, error) {
		return ConfigView{Appdir: "/a", Origin: "default", Lines: []ConfigLine{{Key: "k", Value: "v"}}}, nil
	}
	base.deps.ListDataFiles = func() ([]DataFile, error) {
		return []DataFile{{Name: "header.toml", Path: "/a/data/header.toml"}}, nil
	}
	// config screen
	m2, cmd := step(base, runeKey("6"))
	mc := drain(m2, cmd)
	if !mc.config.loaded || len(mc.config.view.Lines) != 1 {
		t.Fatalf("config not loaded: %+v", mc.config)
	}
	// data screen
	m3, cmd2 := step(base, runeKey("5"))
	md := drain(m3, cmd2)
	if !md.data.loaded || len(md.data.files) != 1 {
		t.Fatalf("data not loaded: %+v", md.data)
	}
}
