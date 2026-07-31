//go:build !notui

package tui

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase/track"
)

func TestFilterApps(t *testing.T) {
	apps := []domain.Application{
		{Company: "Acme", Role: "Go Eng", ID: "2026-07-01_acme_go", Status: domain.StatusApplied},
		{Company: "Beta", Role: "Rust Dev", ID: "2026-07-02_beta_rust", Status: domain.StatusScreen},
	}
	if got := filterApps(apps, ""); len(got) != 2 {
		t.Fatalf("empty query keeps all, got %d", len(got))
	}
	if got := filterApps(apps, "RUST"); len(got) != 1 || got[0].Company != "Beta" {
		t.Fatalf("role match failed: %v", got)
	}
	if got := filterApps(apps, "acme"); len(got) != 1 || got[0].Company != "Acme" {
		t.Fatalf("company match failed: %v", got)
	}
	if got := filterApps(apps, "screen"); len(got) != 1 || got[0].Company != "Beta" {
		t.Fatalf("status match failed: %v", got)
	}
	if got := filterApps(apps, "zzz"); len(got) != 0 {
		t.Fatalf("no-match should be empty, got %d", len(got))
	}
}

func seedNamed(t *testing.T, store *memStore, company, role string) {
	t.Helper()
	tr := &track.Tracker{Store: store, Now: fixedNow, GhostAfterDays: 30}
	if _, err := tr.New(context.Background(), track.NewInput{Company: company, Role: role}); err != nil {
		t.Fatal(err)
	}
}

func TestList_FilterFlow(t *testing.T) {
	store := newMemStore()
	seedNamed(t, store, "Acme", "Go")
	seedNamed(t, store, "Beta", "Rust")
	m := newTestModel(store)
	m = drain(m, m.Init())
	m = press(m, runeKey("2")) // → applications list
	if m.active != screenList || len(m.apps) != 2 {
		t.Fatalf("not on list with 2 apps: active=%d n=%d", m.active, len(m.apps))
	}
	// open filter and type
	m = press(m, runeKey("/"))
	if !m.filtering {
		t.Fatal("'/' should open the filter")
	}
	m = press(m, runeKey("beta"))
	if got := len(m.visibleApps()); got != 1 {
		t.Fatalf("live filter should show 1 match, got %d", got)
	}
	// enter keeps the filter, esc clears it
	m = press(m, namedKey(tea.KeyEnter))
	if m.filtering || m.filter != "beta" {
		t.Fatalf("enter should keep filter: filtering=%v filter=%q", m.filtering, m.filter)
	}
	m = press(m, namedKey(tea.KeyEsc))
	if m.filter != "" {
		t.Fatalf("esc should clear filter, got %q", m.filter)
	}
}

// TestList_FilterCursorSurvivesShrink is the regression test for the v1.5
// recheck's crash: the cursor indexes the FILTERED view, and a reload that
// shrinks the visible set (an action changed an app so it stops matching) must
// clamp the cursor — enter/y on the stale index used to panic.
func TestList_FilterCursorSurvivesShrink(t *testing.T) {
	store := newMemStore()
	seedNamed(t, store, "Acme", "Go")
	seedNamed(t, store, "Beta", "Rust")
	seedNamed(t, store, "Gamma", "C")
	// advance Acme and Beta to applied so the filter matches exactly two
	tr := &track.Tracker{Store: store, Now: fixedNow, GhostAfterDays: 30}
	for id, a := range store.m {
		if a.Company != "Gamma" {
			if _, err := tr.Transition(context.Background(), id, domain.StatusApplied, ""); err != nil {
				t.Fatal(err)
			}
		}
	}

	m := newTestModel(store)
	m = drain(m, m.Init())
	m = press(m, runeKey("2"))
	m = press(m, runeKey("/"))
	m = press(m, runeKey("applied"))
	m = press(m, namedKey(tea.KeyEnter)) // keep filter: vis=2
	m = press(m, runeKey("j"))           // cursor=1 (second match)
	m = press(m, namedKey(tea.KeyEnter)) // detail on it
	m = press(m, runeKey("s"))
	m = press(m, namedKey(tea.KeyEnter)) // transition: it leaves the "applied" filter
	m = press(m, namedKey(tea.KeyEsc))   // back to list: vis shrank to 1
	if got := len(m.visibleApps()); got != 1 {
		t.Fatalf("precondition: vis = %d, want 1", got)
	}
	if m.cursor >= 1 {
		t.Fatalf("cursor not clamped to filtered view: %d", m.cursor)
	}
	m = press(m, namedKey(tea.KeyEnter)) // must not panic
	if m.active != screenDetail {
		t.Fatalf("enter on clamped cursor should open detail, active=%d", m.active)
	}
}

// TestDetail_DoubleEnterDispatchesOnce: a second enter (key auto-repeat) before
// the async result returns must not dispatch a second mutation.
func TestDetail_DoubleEnterDispatchesOnce(t *testing.T) {
	store := newMemStore()
	seedNamed(t, store, "Acme", "Go")
	m := newTestModel(store)
	m = drain(m, m.Init())
	m = press(m, runeKey("2"))
	m = press(m, namedKey(tea.KeyEnter)) // detail
	m = press(m, runeKey("n"))
	m = press(m, runeKey("dup?"))

	// First enter: dispatch WITHOUT draining (result still in flight).
	m2, cmd1 := step(m, namedKey(tea.KeyEnter))
	m = m2
	if cmd1 == nil || !m.detail.busy {
		t.Fatal("first enter should dispatch and mark busy")
	}
	// Second enter while busy: must be a no-op.
	m2, cmd2 := step(m, namedKey(tea.KeyEnter))
	m = m2
	if cmd2 != nil {
		t.Fatal("second enter while busy dispatched a duplicate command")
	}
	// Deliver the first result; busy clears and exactly one note event exists.
	m = drain(m, cmd1)
	if m.detail.busy {
		t.Fatal("busy should clear on actionDoneMsg")
	}
	var app domain.Application
	for _, a := range store.m {
		app = a
	}
	notes := 0
	for _, e := range app.Events {
		if e.Kind == "note" {
			notes++
		}
	}
	if notes != 1 {
		t.Fatalf("want exactly 1 note event, got %d", notes)
	}
}

func TestTruncate_RuneSafe(t *testing.T) {
	got := truncate("Программист-разработчик", 10)
	if len([]rune(got)) != 10 {
		t.Fatalf("want 10 runes, got %d (%q)", len([]rune(got)), got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("want ellipsis suffix, got %q", got)
	}
	for _, r := range got {
		if r == '�' {
			t.Fatalf("mangled UTF-8 in %q", got)
		}
	}
	if s := truncate("short", 24); s != "short" {
		t.Fatalf("short strings pass through, got %q", s)
	}
}

func TestList_YankCopiesID(t *testing.T) {
	store := newMemStore()
	seedNamed(t, store, "Acme", "Go")
	m := newTestModel(store)
	var copied string
	m.deps.Copy = func(_ context.Context, text string) error { copied = text; return nil }
	m = drain(m, m.Init())
	m = press(m, runeKey("2")) // list
	wantID := m.apps[0].ID
	m = press(m, runeKey("y"))
	if copied != wantID {
		t.Fatalf("yank copied %q, want the selected id %q", copied, wantID)
	}
	if m.flash == "" {
		t.Fatal("yank should set a confirmation flash")
	}
}
