//go:build !notui

package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crispuscrew/resumegen/internal/domain"
)

func TestNewApp_CreateFlow(t *testing.T) {
	store := newMemStore()
	m := newTestModel(store)
	m = drain(m, m.Init())

	// `n` on the (empty) dashboard opens the form
	m = press(m, runeKey("n"))
	if m.active != screenNewApp {
		t.Fatalf("'n' should open the creation form, active=%d", m.active)
	}
	m = press(m, runeKey("Acme Corp"))   // company
	m = press(m, namedKey(tea.KeyEnter)) // → role
	m = press(m, runeKey("Go Engineer")) // role
	m = press(m, namedKey(tea.KeyEnter)) // → profile
	m = press(m, runeKey("go-backend"))  // profile
	m = press(m, namedKey(tea.KeyEnter)) // → source
	m = press(m, namedKey(tea.KeyEnter)) // → jd
	m = press(m, namedKey(tea.KeyEnter)) // → salary (last)
	m = press(m, namedKey(tea.KeyEnter)) // submit + drain result

	if m.active != screenDetail {
		t.Fatalf("successful create should open the detail view, active=%d (err %q)", m.active, m.newapp.errMsg)
	}
	if m.detail.app.Company != "Acme Corp" || m.detail.app.Role != "Go Engineer" {
		t.Fatalf("detail shows wrong app: %+v", m.detail.app)
	}
	if m.detail.app.Status != domain.StatusDrafting {
		t.Fatalf("new app should be drafting, got %s", m.detail.app.Status)
	}
	if m.detail.flash == "" {
		t.Fatal("expected a 'created <id>' flash")
	}
	// persisted with the created event, profile stored
	if len(store.m) != 1 {
		t.Fatalf("store should hold exactly 1 app, got %d", len(store.m))
	}
	for _, a := range store.m {
		if a.Profile != "go-backend" || !hasEvent(a, "created") {
			t.Fatalf("stored app wrong: %+v", a)
		}
	}
}

func TestNewApp_MissingRequiredShowsError(t *testing.T) {
	store := newMemStore()
	m := newTestModel(store)
	m = drain(m, m.Init())
	m = press(m, runeKey("2")) // list
	m = press(m, runeKey("n"))
	// submit straight through the empty form
	for i := 0; i < len(newAppFields); i++ {
		m = press(m, namedKey(tea.KeyEnter))
	}
	if m.active != screenNewApp {
		t.Fatalf("validation failure must keep the form open, active=%d", m.active)
	}
	if m.newapp.errMsg == "" {
		t.Fatal("expected a company/role required error")
	}
	if len(store.m) != 0 {
		t.Fatalf("nothing should be created, store has %d", len(store.m))
	}
}

func TestNewApp_EscCancelsWithoutCreating(t *testing.T) {
	store := newMemStore()
	m := newTestModel(store)
	m = drain(m, m.Init())
	m = press(m, runeKey("2"))
	m = press(m, runeKey("n"))
	m = press(m, runeKey("half-typed"))
	m = press(m, namedKey(tea.KeyEsc))
	if m.active != screenList {
		t.Fatalf("esc should return to the list, active=%d", m.active)
	}
	if len(store.m) != 0 {
		t.Fatalf("esc must not create, store has %d", len(store.m))
	}
}

func TestNewApp_TypedQAndDigitsStayInForm(t *testing.T) {
	// the form captures all keys: q and digits must be typed, not act globally
	m := newTestModel(newMemStore())
	m = drain(m, m.Init())
	m = press(m, runeKey("n"))
	m = press(m, runeKey("q1w2"))
	if m.quitting {
		t.Fatal("typing 'q' in the form must not quit")
	}
	if m.active != screenNewApp {
		t.Fatalf("digits must not switch screens, active=%d", m.active)
	}
	if got := m.newapp.inputs[0].Value(); got != "q1w2" {
		t.Fatalf("field should contain the typed text, got %q", got)
	}
}
