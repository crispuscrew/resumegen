package track

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// memStore is an in-memory track.Store for deterministic use-case tests.
type memStore struct {
	m map[string]domain.Application
}

func newMemStore() *memStore { return &memStore{m: map[string]domain.Application{}} }

func (s *memStore) Save(_ context.Context, a domain.Application) error {
	s.m[a.ID] = a
	return nil
}
func (s *memStore) Load(_ context.Context, id string) (domain.Application, error) {
	a, ok := s.m[id]
	if !ok {
		return domain.Application{}, errors.New("not found: " + id)
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

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

func newTracker(store Store, now time.Time) *Tracker {
	return &Tracker{Store: store, Now: fixedClock(now), GhostAfterDays: 30}
}

var testNow = time.Date(2026, time.July, 8, 9, 30, 0, 0, time.UTC)

func TestNew_BuildsDraftingWithCreatedEvent(t *testing.T) {
	ctx := context.Background()
	tr := newTracker(newMemStore(), testNow)
	app, err := tr.New(ctx, NewInput{Company: "Acme Corp", Role: "Senior Go"})
	if err != nil {
		t.Fatal(err)
	}
	if app.ID != "2026-07-08_acme-corp_senior-go" {
		t.Errorf("id = %q", app.ID)
	}
	if app.Status != domain.StatusDrafting {
		t.Errorf("status = %q, want drafting", app.Status)
	}
	if len(app.Events) != 1 || app.Events[0].Kind != "created" {
		t.Errorf("want one created event, got %+v", app.Events)
	}
	if !app.AppliedAt.IsZero() {
		// a drafting entry has not been applied; the date is stamped only on
		// the actual transition to applied
		t.Errorf("applied_at should be zero for drafting, got %v", app.AppliedAt)
	}
}

func TestNew_RequiresCompanyAndRole(t *testing.T) {
	ctx := context.Background()
	tr := newTracker(newMemStore(), testNow)
	if _, err := tr.New(ctx, NewInput{Role: "x"}); err == nil {
		t.Error("missing company should error")
	}
	if _, err := tr.New(ctx, NewInput{Company: "x"}); err == nil {
		t.Error("missing role should error")
	}
}

func TestNew_IDCollisionSuffix(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	tr := newTracker(store, testNow)
	a1, _ := tr.New(ctx, NewInput{Company: "Acme", Role: "Go"})
	a2, _ := tr.New(ctx, NewInput{Company: "Acme", Role: "Go"})
	a3, _ := tr.New(ctx, NewInput{Company: "Acme", Role: "Go"})
	if a1.ID != "2026-07-08_acme_go" {
		t.Errorf("a1 = %q", a1.ID)
	}
	if a2.ID != "2026-07-08_acme_go-2" {
		t.Errorf("a2 = %q", a2.ID)
	}
	if a3.ID != "2026-07-08_acme_go-3" {
		t.Errorf("a3 = %q", a3.ID)
	}
}

func TestNew_StatusShortcutAdvances(t *testing.T) {
	ctx := context.Background()
	tr := newTracker(newMemStore(), testNow)
	app, err := tr.New(ctx, NewInput{Company: "Acme", Role: "Go", Status: domain.StatusScreen})
	if err != nil {
		t.Fatal(err)
	}
	if app.Status != domain.StatusScreen {
		t.Errorf("status = %q, want screen", app.Status)
	}
	// created + applied + screen
	if len(app.Events) != 3 {
		t.Errorf("want 3 events (created, applied, screen), got %d: %+v", len(app.Events), app.Events)
	}
	if !app.AppliedAt.Equal(day(2026, time.July, 8)) {
		t.Errorf("applied_at should be set when passing applied, got %v", app.AppliedAt)
	}
}

func TestNew_StatusShortcutRejectsTerminalTargetFromDrafting(t *testing.T) {
	ctx := context.Background()
	tr := newTracker(newMemStore(), testNow)
	// rejected is unreachable from drafting even via the shortcut
	if _, err := tr.New(ctx, NewInput{Company: "Acme", Role: "Go", Status: domain.StatusRejected}); err == nil {
		t.Error("drafting -> rejected shortcut should error")
	}
	// unknown status is an error
	if _, err := tr.New(ctx, NewInput{Company: "Acme", Role: "Go", Status: domain.Status("bogus")}); err == nil {
		t.Error("unknown status should error")
	}
}

func TestNew_RejectsGhostedShortcut(t *testing.T) {
	ctx := context.Background()
	tr := newTracker(newMemStore(), testNow)
	// ghosted is auto-only; it cannot be created by hand
	if _, err := tr.New(ctx, NewInput{Company: "Acme", Role: "Go", Status: domain.StatusGhosted}); err == nil {
		t.Error("creating an application as ghosted should be rejected")
	}
}

func TestTransition_ManualGhostRejectedButWithdrawnAllowed(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	tr := newTracker(store, testNow)
	app, err := tr.New(ctx, NewInput{Company: "Acme", Role: "Go", Status: domain.StatusApplied})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tr.Transition(ctx, app.ID, domain.StatusGhosted, ""); err == nil {
		t.Error("manual transition to ghosted should be rejected (auto-only)")
	}
	if _, err := tr.Transition(ctx, app.ID, domain.StatusWithdrawn, "changed my mind"); err != nil {
		t.Errorf("withdrawn is user-driven and should be allowed from an active state: %v", err)
	}
}

func TestGet_LazyAutoGhosts(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	created := newTracker(store, testNow)
	app, err := created.New(ctx, NewInput{Company: "Acme", Role: "Go", Status: domain.StatusApplied})
	if err != nil {
		t.Fatal(err)
	}

	// Read 40 days later: past the 30-day threshold with no activity -> auto-ghost.
	later := testNow.Add(40 * 24 * time.Hour)
	tr := newTracker(store, later)
	got, err := tr.Get(ctx, app.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusGhosted {
		t.Fatalf("want ghosted after 40 idle days, got %q", got.Status)
	}
	if reloaded, _ := store.Load(ctx, app.ID); reloaded.Status != domain.StatusGhosted {
		t.Error("auto-ghost must be persisted to the store")
	}
	// Idempotent: a second read appends no further events.
	before := len(got.Events)
	got2, _ := tr.Get(ctx, app.ID)
	if len(got2.Events) != before {
		t.Errorf("second read should not append events: %d -> %d", before, len(got2.Events))
	}

	// A fresh active entry read soon after creation is not ghosted.
	fresh := newTracker(store, testNow)
	app2, _ := fresh.New(ctx, NewInput{Company: "Beta", Role: "Go", Status: domain.StatusApplied})
	if g, _ := fresh.Get(ctx, app2.ID); g.Status != domain.StatusApplied {
		t.Errorf("fresh entry should not ghost, got %q", g.Status)
	}
}

func TestList_Filters(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	tr := newTracker(store, testNow)
	a, _ := tr.New(ctx, NewInput{Company: "Acme", Role: "Go", Status: domain.StatusApplied})
	_, _ = tr.New(ctx, NewInput{Company: "Beta", Role: "Rust", Status: domain.StatusScreen})
	if _, err := tr.AddFollowup(ctx, a.ID, testNow.Add(-24*time.Hour), "nudge"); err != nil {
		t.Fatal(err)
	}

	// status filter (OR within the set)
	if got, _ := tr.List(ctx, Query{Statuses: []domain.Status{domain.StatusScreen}}); len(got) != 1 || got[0].Status != domain.StatusScreen {
		t.Errorf("status filter: got %v", ids(got))
	}
	// followups-due: only `a` has an overdue undone followup
	if got, _ := tr.List(ctx, Query{FollowupsDue: true}); len(got) != 1 || got[0].ID != a.ID {
		t.Errorf("followups-due filter: got %v", ids(got))
	}
	// stale: nothing is stale immediately
	if got, _ := tr.List(ctx, Query{}.WithStaleDays(5)); len(got) != 0 {
		t.Errorf("nothing stale at creation, got %v", ids(got))
	}
	// stale 10 days later (still under the 30-day ghost threshold, so no ghosting)
	lateTr := newTracker(store, testNow.Add(10*24*time.Hour))
	if got, _ := lateTr.List(ctx, Query{}.WithStaleDays(5)); len(got) != 2 {
		t.Errorf("both stale after 10 idle days, got %v", ids(got))
	}
}

func ids(apps []domain.Application) []string {
	out := make([]string, len(apps))
	for i, a := range apps {
		out[i] = a.ID
	}
	return out
}

func TestTransition_AppendsEventAndSaves(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	tr := newTracker(store, testNow)
	app, _ := tr.New(ctx, NewInput{Company: "Acme", Role: "Go"})

	got, err := tr.Transition(ctx, app.ID, domain.StatusApplied, "sent via referral")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusApplied {
		t.Errorf("status = %q", got.Status)
	}
	last := got.Events[len(got.Events)-1]
	if last.Kind != "status" || last.Note != "applied: sent via referral" {
		t.Errorf("last event = %+v", last)
	}
	// persisted
	reloaded, _ := store.Load(ctx, app.ID)
	if reloaded.Status != domain.StatusApplied {
		t.Errorf("not persisted: %q", reloaded.Status)
	}
}

func TestTransition_InvalidReturnsError(t *testing.T) {
	ctx := context.Background()
	tr := newTracker(newMemStore(), testNow)
	app, _ := tr.New(ctx, NewInput{Company: "Acme", Role: "Go"})
	// drafting -> interview is a skip; must error
	_, err := tr.Transition(ctx, app.ID, domain.StatusInterview, "")
	var ite *domain.InvalidTransitionError
	if !errors.As(err, &ite) {
		t.Errorf("want *InvalidTransitionError, got %v", err)
	}
}

func TestTransition_UnknownID(t *testing.T) {
	ctx := context.Background()
	tr := newTracker(newMemStore(), testNow)
	if _, err := tr.Transition(ctx, "nope", domain.StatusApplied, ""); err == nil {
		t.Error("unknown id should error")
	}
}

func TestAddFollowup(t *testing.T) {
	ctx := context.Background()
	store := newMemStore()
	tr := newTracker(store, testNow)
	app, _ := tr.New(ctx, NewInput{Company: "Acme", Role: "Go"})
	due := day(2026, time.July, 20)
	got, err := tr.AddFollowup(ctx, app.ID, due, "ping recruiter")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Followups) != 1 || got.Followups[0].Action != "ping recruiter" {
		t.Errorf("followups = %+v", got.Followups)
	}
	if !got.Followups[0].Due.Equal(due) {
		t.Errorf("due = %v", got.Followups[0].Due)
	}
	if got.Events[len(got.Events)-1].Kind != "followup" {
		t.Errorf("want followup event, got %+v", got.Events)
	}
}

func TestAddFollowup_RequiresAction(t *testing.T) {
	ctx := context.Background()
	tr := newTracker(newMemStore(), testNow)
	app, _ := tr.New(ctx, NewInput{Company: "Acme", Role: "Go"})
	if _, err := tr.AddFollowup(ctx, app.ID, testNow, "  "); err == nil {
		t.Error("empty action should error")
	}
}

func TestAddNote_DraftingEditsNotesInPlace(t *testing.T) {
	ctx := context.Background()
	tr := newTracker(newMemStore(), testNow)
	app, _ := tr.New(ctx, NewInput{Company: "Acme", Role: "Go"})
	got, err := tr.AddNote(ctx, app.ID, "first note")
	if err != nil {
		t.Fatal(err)
	}
	if got.Notes != "first note" {
		t.Errorf("notes = %q", got.Notes)
	}
	got, _ = tr.AddNote(ctx, app.ID, "second note")
	if got.Notes != "first note\nsecond note" {
		t.Errorf("notes = %q", got.Notes)
	}
	// each note also appends an event
	notes := 0
	for _, e := range got.Events {
		if e.Kind == "note" {
			notes++
		}
	}
	if notes != 2 {
		t.Errorf("want 2 note events, got %d", notes)
	}
}

func TestAddNote_NonDraftingDoesNotEditNotes(t *testing.T) {
	ctx := context.Background()
	tr := newTracker(newMemStore(), testNow)
	app, _ := tr.New(ctx, NewInput{Company: "Acme", Role: "Go", Status: domain.StatusApplied})
	got, err := tr.AddNote(ctx, app.ID, "post-apply note")
	if err != nil {
		t.Fatal(err)
	}
	if got.Notes != "" {
		t.Errorf("applied entry should not edit Notes in place, got %q", got.Notes)
	}
	if got.Events[len(got.Events)-1].Kind != "note" {
		t.Error("note event should still be appended")
	}
}

func TestAddNote_RequiresText(t *testing.T) {
	ctx := context.Background()
	tr := newTracker(newMemStore(), testNow)
	app, _ := tr.New(ctx, NewInput{Company: "Acme", Role: "Go"})
	if _, err := tr.AddNote(ctx, app.ID, ""); err == nil {
		t.Error("empty note should error")
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Acme Corp":         "acme-corp",
		"Senior Go/Backend": "senior-go-backend",
		"  spaced  ":        "spaced",
		"C++ Dev":           "c-dev",
		"!!!":               "x",
		"":                  "x",
		"already-slug":      "already-slug",
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSet_MultiStepAdvance(t *testing.T) {
	tr := newTracker(newMemStore(), testNow)
	app, _ := tr.New(context.Background(), NewInput{Company: "A", Role: "B", Status: domain.StatusApplied})
	// applied -> offer skips screen+interview: each hop must be evented
	got, err := tr.Set(context.Background(), app.ID, domain.StatusOffer, "fast track")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusOffer {
		t.Fatalf("status = %s", got.Status)
	}
	statuses := []string{}
	for _, e := range got.Events {
		if e.Kind == "status" {
			statuses = append(statuses, e.Note)
		}
	}
	// applied (from New), then screen, interview, offer; note only on the last hop
	want := []string{"applied", "screen", "interview", "offer: fast track"}
	if len(statuses) != len(want) {
		t.Fatalf("status events = %v, want %v", statuses, want)
	}
	for i := range want {
		if statuses[i] != want[i] {
			t.Fatalf("status events = %v, want %v", statuses, want)
		}
	}
	// backwards is still invalid
	if _, err := tr.Set(context.Background(), app.ID, domain.StatusScreen, ""); err == nil {
		t.Error("backwards Set must fail")
	}
	// manual ghosted still blocked
	if _, err := tr.Set(context.Background(), app.ID, domain.StatusGhosted, ""); err == nil {
		t.Error("manual ghosted must stay blocked")
	}
}

func TestCompleteFollowup(t *testing.T) {
	tr := newTracker(newMemStore(), testNow)
	app, _ := tr.New(context.Background(), NewInput{Company: "A", Role: "B"})
	_, _ = tr.AddFollowup(context.Background(), app.ID, testNow, "ping them")
	got, err := tr.CompleteFollowup(context.Background(), app.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Followups[0].Done {
		t.Fatal("followup not marked done")
	}
	found := false
	for _, e := range got.Events {
		if e.Kind == "followup-done" && e.Note == "ping them" {
			found = true
		}
	}
	if !found {
		t.Fatal("no followup-done event")
	}
	if _, err := tr.CompleteFollowup(context.Background(), app.ID, 1); err == nil {
		t.Error("completing twice must error")
	}
	if _, err := tr.CompleteFollowup(context.Background(), app.ID, 5); err == nil {
		t.Error("out-of-range index must error")
	}
}

func TestEdit(t *testing.T) {
	tr := newTracker(newMemStore(), testNow)
	app, _ := tr.New(context.Background(), NewInput{Company: "Acme", Role: "Go"})
	nc, remote := "Acme GmbH", true
	got, err := tr.Edit(context.Background(), app.ID, EditInput{Company: &nc, Remote: &remote})
	if err != nil {
		t.Fatal(err)
	}
	if got.Company != "Acme GmbH" || !got.Remote {
		t.Fatalf("edit not applied: %+v", got)
	}
	if got.ID != app.ID {
		t.Fatalf("ID must be stable, got %s", got.ID)
	}
	edited := false
	for _, e := range got.Events {
		if e.Kind == "edited" && e.Note == "company, remote" {
			edited = true
		}
	}
	if !edited {
		t.Fatalf("edited event missing/wrong: %+v", got.Events)
	}
	empty := "  "
	if _, err := tr.Edit(context.Background(), app.ID, EditInput{Company: &empty}); err == nil {
		t.Error("clearing company must error")
	}
	if _, err := tr.Edit(context.Background(), app.ID, EditInput{}); err == nil {
		t.Error("no-op edit must error")
	}
}

func TestDeleteAndReopen(t *testing.T) {
	store := newMemStore()
	tr := newTracker(store, testNow)
	// reopen: submitted -> withdrawn -> back to applied
	a1, _ := tr.New(context.Background(), NewInput{Company: "A", Role: "R", Status: domain.StatusApplied})
	_, _ = tr.Set(context.Background(), a1.ID, domain.StatusWithdrawn, "")
	got, err := tr.Reopen(context.Background(), a1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusApplied {
		t.Fatalf("reopen of a submitted app should land on applied, got %s", got.Status)
	}
	// reopen an active app must fail
	if _, err := tr.Reopen(context.Background(), a1.ID); err == nil {
		t.Error("reopening an active app must error")
	}
	// delete removes it
	if err := tr.Delete(context.Background(), a1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.Get(context.Background(), a1.ID); err == nil {
		t.Error("deleted app still loads")
	}
	if err := tr.Delete(context.Background(), "nope"); err == nil {
		t.Error("deleting a missing id must error")
	}
}

func TestAddContact(t *testing.T) {
	tr := newTracker(newMemStore(), testNow)
	app, _ := tr.New(context.Background(), NewInput{Company: "A", Role: "R"})
	got, err := tr.AddContact(context.Background(), app.ID, domain.AppContact{Name: "Ivan", Role: "recruiter"})
	if err != nil || len(got.Contacts) != 1 || got.Contacts[0].Name != "Ivan" {
		t.Fatalf("contact not recorded: %+v err %v", got.Contacts, err)
	}
	if _, err := tr.AddContact(context.Background(), app.ID, domain.AppContact{}); err == nil {
		t.Error("contact without a name must error")
	}
}

// --- timezone regression -----------------------------------------------------
//
// Every other test in this file uses a UTC clock, which hid a day-off-by-one:
// DayFloor used to read the calendar day from t.UTC() instead of from t's own
// location, so a local clock east or west of UTC produced the wrong "today"
// for a window equal to the offset.

// tzClock returns a clock reading wall-time y-m-d h:00 in a fixed offset zone.
func tzClock(y int, mo time.Month, d, h, offsetHours int) func() time.Time {
	zone := time.FixedZone("TEST", offsetHours*3600)
	return fixedClock(time.Date(y, mo, d, h, 0, 0, 0, zone))
}

func TestNew_IDUsesLocalCalendarDay(t *testing.T) {
	for _, tc := range []struct {
		name        string
		h, offset   int
		wantID      string
		wantApplied string
	}{
		// 18:00 on the 25th at UTC-7 is 01:00 on the 26th in UTC.
		{"west of UTC", 18, -7, "2026-07-25_acme_go", "2026-07-25"},
		// 01:00 on the 26th at UTC+3 (MSK) is 22:00 on the 25th in UTC.
		{"east of UTC", 1, +3, "2026-07-26_acme_go", "2026-07-26"},
		{"at UTC", 12, 0, "2026-07-26_acme_go", "2026-07-26"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			day := 25
			if tc.offset >= 0 {
				day = 26
			}
			tr := &Tracker{Store: newMemStore(), Now: tzClock(2026, time.July, day, tc.h, tc.offset), GhostAfterDays: 30}
			app, err := tr.New(context.Background(), NewInput{Company: "Acme", Role: "Go", Status: domain.StatusApplied})
			if err != nil {
				t.Fatal(err)
			}
			if app.ID != tc.wantID {
				t.Errorf("ID = %q, want %q", app.ID, tc.wantID)
			}
			if got := app.AppliedAt.Format("2006-01-02"); got != tc.wantApplied {
				t.Errorf("applied_at = %s, want %s", got, tc.wantApplied)
			}
		})
	}
}

func TestAddFollowup_DefaultDueIsLocalTodayPlusLag(t *testing.T) {
	// 01:00 MSK on the 26th: the UTC day is still the 25th. With a 7-day lag the
	// due date must be the 26th + 7 = Aug 2, not Aug 1.
	tr := &Tracker{
		Store: newMemStore(), Now: tzClock(2026, time.July, 26, 1, +3),
		GhostAfterDays: 30, FollowupLagDays: 7,
	}
	ctx := context.Background()
	app, err := tr.New(ctx, NewInput{Company: "Acme", Role: "Go"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := tr.AddFollowup(ctx, app.ID, time.Time{}, "ping recruiter")
	if err != nil {
		t.Fatal(err)
	}
	if due := got.Followups[0].Due.Format("2006-01-02"); due != "2026-08-02" {
		t.Errorf("default followup due = %s, want 2026-08-02 (local today + 7)", due)
	}
}

func TestHasDueFollowup_DoesNotFireADayEarly(t *testing.T) {
	// 18:00 on the 25th at UTC-7 (= 01:00 on the 26th UTC). A followup due on
	// the 26th is tomorrow locally and must not count as due yet.
	now := time.Date(2026, time.July, 25, 18, 0, 0, 0, time.FixedZone("TEST", -7*3600))
	app := domain.Application{Followups: []domain.Followup{{Due: day(2026, time.July, 26)}}}
	if app.HasDueFollowup(now) {
		t.Error("HasDueFollowup = true for a followup due tomorrow (local), want false")
	}
	if !app.HasDueFollowup(now.AddDate(0, 0, 1)) {
		t.Error("HasDueFollowup = false on the due day itself, want true")
	}
}
