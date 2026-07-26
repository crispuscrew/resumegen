//go:build !notui

package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGenerate_RenderSuccess(t *testing.T) {
	m := newTestModel(newMemStore())
	m.deps.ListProfiles = func() ([]string, error) { return []string{"default", "cpp"}, nil }
	m.deps.Render = func(_ context.Context, profile string) (string, error) {
		return "/out/" + profile + ".pdf", nil
	}
	// enter generate + load profiles
	m2, cmd := step(m, runeKey("3"))
	m = drain(m2, cmd)
	if m.active != screenGenerate {
		t.Fatalf("expected generate screen, got %d", m.active)
	}
	if len(m.generate.profiles) != 2 {
		t.Fatalf("profiles not loaded: %v", m.generate.profiles)
	}
	// render the selected (first) profile
	m, cmd = step(m, namedKey(tea.KeyEnter))
	if m.generate.phase != genRunning {
		t.Fatalf("phase should be running, got %d", m.generate.phase)
	}
	m = drain(m, cmd) // render returns immediately
	if m.generate.phase != genDone {
		t.Fatalf("phase should be done, got %d", m.generate.phase)
	}
	if m.generate.outPath != "/out/default.pdf" {
		t.Fatalf("outPath = %q", m.generate.outPath)
	}
}

// TestGenerate_CancelMidRender proves esc cancels an in-flight render: the fake
// renderer blocks until its context is cancelled (exactly how the real typst
// exec.CommandContext dies), and the model returns to a non-running state with no
// output. This is SPEC acceptance #2.
func TestGenerate_CancelMidRender(t *testing.T) {
	started := make(chan struct{})
	m := newTestModel(newMemStore())
	m.deps.ListProfiles = func() ([]string, error) { return []string{"default"}, nil }
	m.deps.Render = func(ctx context.Context, _ string) (string, error) {
		close(started)
		<-ctx.Done() // block until cancelled, like a killed subprocess
		return "", ctx.Err()
	}
	m2, cmd := step(m, runeKey("3"))
	m = drain(m2, cmd)

	// begin the render; capture the (blocking) command to run off the main path
	m, cmd = step(m, namedKey(tea.KeyEnter))
	if m.generate.phase != genRunning {
		t.Fatalf("phase should be running, got %d", m.generate.phase)
	}
	msgCh := make(chan tea.Msg, 1)
	go func() { msgCh <- cmd() }()
	<-started // the render is in flight

	// esc must cancel it (isCapturing locks the screen so esc reaches the handler)
	m, _ = step(m, namedKey(tea.KeyEsc))

	var done tea.Msg
	select {
	case done = <-msgCh:
	case <-time.After(2 * time.Second):
		t.Fatal("render did not cancel within 2s")
	}
	m, _ = step(m, done)
	if m.generate.phase == genRunning {
		t.Fatal("still running after cancel")
	}
	if m.generate.outPath != "" {
		t.Fatalf("cancel must not produce output, got %q", m.generate.outPath)
	}
}

// TestGenerate_LockedWhileRendering asserts a digit key cannot navigate away
// mid-render (which would orphan the running typst).
func TestGenerate_LockedWhileRendering(t *testing.T) {
	m := newTestModel(newMemStore())
	m.deps.ListProfiles = func() ([]string, error) { return []string{"default"}, nil }
	m.deps.Render = func(ctx context.Context, _ string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}
	m2, cmd := step(m, runeKey("3"))
	m = drain(m2, cmd)
	m, _ = step(m, namedKey(tea.KeyEnter)) // start render (don't run the cmd)
	// try to jump to config with "6"
	m, _ = step(m, runeKey("6"))
	if m.active != screenGenerate {
		t.Fatalf("digit nav should be blocked mid-render, active=%d", m.active)
	}
	if m.generate.cancel != nil {
		m.generate.cancel() // clean up the dangling context
	}
}
