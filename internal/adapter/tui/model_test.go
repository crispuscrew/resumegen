//go:build !notui

package tui

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase/track"
)

// memStore is an in-memory track.Store so the model can be driven deterministically
// without touching the filesystem. It mirrors the trackrepo contract.
type memStore struct{ m map[string]domain.Application }

func newMemStore() *memStore { return &memStore{m: map[string]domain.Application{}} }

func (s *memStore) Save(_ context.Context, a domain.Application) error {
	s.m[a.ID] = a
	return nil
}

func (s *memStore) Load(_ context.Context, id string) (domain.Application, error) {
	a, ok := s.m[id]
	if !ok {
		return domain.Application{}, fmt.Errorf("no application %q", id)
	}
	return a, nil
}

func (s *memStore) List(_ context.Context) ([]domain.Application, error) {
	out := make([]domain.Application, 0, len(s.m))
	for _, a := range s.m {
		out = append(out, a)
	}
	return out, nil
}

func (s *memStore) Exists(_ context.Context, id string) (bool, error) {
	_, ok := s.m[id]
	return ok, nil
}

func (s *memStore) Delete(_ context.Context, id string) error {
	if _, ok := s.m[id]; !ok {
		return fmt.Errorf("no application %q", id)
	}
	delete(s.m, id)
	return nil
}

func fixedNow() time.Time { return time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC) }

func newTestModel(store track.Store) model {
	tr := &track.Tracker{Store: store, Now: fixedNow, GhostAfterDays: 30}
	return model{
		deps: Deps{
			Ctx: context.Background(), Version: "test", Tracker: tr,
			StaleAfterDays: 30, FollowupLagDays: 7,
		},
		styl:    newStyles("default"),
		active:  screenDashboard,
		loading: true,
	}
}

// step applies one message and returns the concrete model + any command.
func step(m model, msg tea.Msg) (model, tea.Cmd) {
	tm, cmd := m.Update(msg)
	return tm.(model), cmd
}

// drain executes a chain of single-message commands (loadApps -> appsLoadedMsg,
// transitionCmd -> actionDoneMsg -> loadApps -> …) until the model settles. Slice 1
// never batches, so a linear loop is sufficient.
func drain(m model, cmd tea.Cmd) model {
	for i := 0; cmd != nil && i < 20; i++ {
		var next tea.Cmd
		m, next = step(m, cmd())
		cmd = next
	}
	return m
}

func runeKey(s string) tea.KeyMsg       { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }
func namedKey(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

func press(m model, k tea.KeyMsg) model {
	m, cmd := step(m, k)
	return drain(m, cmd)
}

func seedDrafting(t *testing.T, store *memStore) domain.Application {
	t.Helper()
	tr := &track.Tracker{Store: store, Now: fixedNow, GhostAfterDays: 30}
	a, err := tr.New(context.Background(), track.NewInput{Company: "Acme", Role: "Go Eng"})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return a
}

func TestModel_LoadsAndNavigates(t *testing.T) {
	store := newMemStore()
	seedDrafting(t, store)
	m := newTestModel(store)
	m = drain(m, m.Init()) // load apps

	if m.loading {
		t.Fatal("still loading after Init drain")
	}
	if len(m.apps) != 1 {
		t.Fatalf("want 1 app loaded, got %d", len(m.apps))
	}
	// dashboard -> enter -> list
	m = press(m, namedKey(tea.KeyEnter))
	if m.active != screenList {
		t.Fatalf("enter on dashboard should open list, active=%d", m.active)
	}
	// list -> enter -> detail
	m = press(m, namedKey(tea.KeyEnter))
	if m.active != screenDetail {
		t.Fatalf("enter on list should open detail, active=%d", m.active)
	}
	if m.detail.app.Company != "Acme" {
		t.Fatalf("detail app not set: %+v", m.detail.app)
	}
}

func TestModel_StatusTransitionAppendsEvent(t *testing.T) {
	store := newMemStore()
	app := seedDrafting(t, store)
	m := newTestModel(store)
	m = drain(m, m.Init())
	m = press(m, namedKey(tea.KeyEnter)) // list
	m = press(m, namedKey(tea.KeyEnter)) // detail

	// open status picker: drafting -> [applied, withdrawn]
	m = press(m, runeKey("s"))
	if m.detail.mode != detailStatusPick {
		t.Fatalf("'s' should open status picker, mode=%d", m.detail.mode)
	}
	if len(m.detail.targets) == 0 || m.detail.targets[0] != domain.StatusApplied {
		t.Fatalf("first target should be applied, got %v", m.detail.targets)
	}
	// pick the first (applied) and apply
	m = press(m, namedKey(tea.KeyEnter))

	if m.detail.app.Status != domain.StatusApplied {
		t.Fatalf("status should be applied after transition, got %s", m.detail.app.Status)
	}
	// the store must reflect it, with a "status" event appended
	saved := store.m[app.ID]
	if saved.Status != domain.StatusApplied {
		t.Fatalf("store not updated, status=%s", saved.Status)
	}
	if !hasEvent(saved, "status") {
		t.Fatalf("no status event appended: %+v", saved.Events)
	}
	// mode returned to view and a flash was set
	if m.detail.mode != detailView || m.detail.flash == "" {
		t.Fatalf("expected view mode + flash, mode=%d flash=%q", m.detail.mode, m.detail.flash)
	}
}

func TestModel_AddNoteThroughUseCase(t *testing.T) {
	store := newMemStore()
	app := seedDrafting(t, store)
	m := newTestModel(store)
	m = drain(m, m.Init())
	m = press(m, namedKey(tea.KeyEnter)) // list
	m = press(m, namedKey(tea.KeyEnter)) // detail

	m = press(m, runeKey("n")) // enter note mode
	if m.detail.mode != detailNote {
		t.Fatalf("'n' should enter note mode, mode=%d", m.detail.mode)
	}
	m = press(m, runeKey("call recruiter")) // type into the input
	m = press(m, namedKey(tea.KeyEnter))    // submit

	saved := store.m[app.ID]
	if !hasEvent(saved, "note") {
		t.Fatalf("no note event appended: %+v", saved.Events)
	}
	// drafting entry also edits Notes in place (use-case behavior, surfaced via TUI)
	if saved.Notes != "call recruiter" {
		t.Fatalf("Notes not set on drafting entry, got %q", saved.Notes)
	}
	if m.detail.mode != detailView {
		t.Fatalf("should return to view after note, mode=%d", m.detail.mode)
	}
}

func TestModel_EscCancelsNoteWithoutMutation(t *testing.T) {
	store := newMemStore()
	app := seedDrafting(t, store)
	before := len(store.m[app.ID].Events)
	m := newTestModel(store)
	m = drain(m, m.Init())
	m = press(m, namedKey(tea.KeyEnter))
	m = press(m, namedKey(tea.KeyEnter))
	m = press(m, runeKey("n"))
	m = press(m, runeKey("half-typed"))
	m = press(m, namedKey(tea.KeyEsc)) // cancel

	if m.detail.mode != detailView {
		t.Fatalf("esc should return to view, mode=%d", m.detail.mode)
	}
	if got := len(store.m[app.ID].Events); got != before {
		t.Fatalf("esc must not mutate: events %d -> %d", before, got)
	}
}

func hasEvent(a domain.Application, kind string) bool {
	for _, e := range a.Events {
		if e.Kind == kind {
			return true
		}
	}
	return false
}
