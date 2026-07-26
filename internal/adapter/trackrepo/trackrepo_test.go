package trackrepo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/crispuscrew/resumegen/internal/domain"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func sampleApp() domain.Application {
	return domain.Application{
		ID:        "2026-07-08_acme_go",
		Company:   "Acme",
		Role:      "Go",
		Profile:   "default",
		Status:    domain.StatusApplied,
		Remote:    true,
		AppliedAt: day(2026, time.July, 8),
		Notes:     "hot lead",
		Contacts: []domain.AppContact{
			{Name: "Ada", Role: "Recruiter", Channel: "email", LastContact: day(2026, time.July, 9)},
		},
		Followups: []domain.Followup{
			{Due: day(2026, time.July, 20), Action: "ping", Done: false},
		},
		Events: []domain.Event{
			{At: time.Date(2026, time.July, 8, 9, 30, 0, 0, time.UTC), Kind: "created"},
			{At: time.Date(2026, time.July, 8, 10, 0, 0, 0, time.UTC), Kind: "status", Note: "applied"},
		},
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	ctx := context.Background()
	in := sampleApp()

	if err := store.Save(ctx, in); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load(ctx, in.ID)
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != in.ID || got.Company != in.Company || got.Status != in.Status || got.Remote != in.Remote {
		t.Errorf("scalar mismatch: %+v", got)
	}
	if !got.AppliedAt.Equal(in.AppliedAt) {
		t.Errorf("applied_at drift: got %v want %v", got.AppliedAt, in.AppliedAt)
	}
	if len(got.Followups) != 1 || !got.Followups[0].Due.Equal(in.Followups[0].Due) {
		t.Errorf("followup due drift: %+v", got.Followups)
	}
	if len(got.Contacts) != 1 || !got.Contacts[0].LastContact.Equal(in.Contacts[0].LastContact) {
		t.Errorf("contact date drift: %+v", got.Contacts)
	}
	if len(got.Events) != 2 || !got.Events[0].At.Equal(in.Events[0].At) {
		t.Errorf("event at drift: %+v", got.Events)
	}
}

func TestSave_DateTypesOnDisk(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	if err := store.Save(context.Background(), sampleApp()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "applications", "2026-07-08_acme_go.toml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	// applied_at is a bare local-date (no T, no timezone)
	if !strings.Contains(text, "applied_at = 2026-07-08") {
		t.Errorf("applied_at should be a local-date; file:\n%s", text)
	}
	// due is a bare local-date
	if !strings.Contains(text, "due = 2026-07-20") {
		t.Errorf("due should be a local-date; file:\n%s", text)
	}
	// event at is an offset-datetime (has T and Z)
	if !strings.Contains(text, "at = 2026-07-08T09:30:00Z") {
		t.Errorf("event at should be offset-datetime; file:\n%s", text)
	}
}

func TestList_SortedAndTolerant(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	ctx := context.Background()
	// empty (no applications/ dir) is not an error
	got, err := store.List(ctx)
	if err != nil || len(got) != 0 {
		t.Fatalf("empty list: got %d err %v", len(got), err)
	}
	a := sampleApp()
	b := a
	b.ID = "2026-07-01_beta_dev"
	if err := store.Save(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, b); err != nil {
		t.Fatal(err)
	}
	got, err = store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "2026-07-01_beta_dev" {
		t.Errorf("want sorted [beta, acme], got %v", ids(got))
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	ctx := context.Background()
	ok, err := store.Exists(ctx, "missing")
	if err != nil || ok {
		t.Errorf("missing: ok=%v err=%v", ok, err)
	}
	if err := store.Save(ctx, sampleApp()); err != nil {
		t.Fatal(err)
	}
	ok, err = store.Exists(ctx, "2026-07-08_acme_go")
	if err != nil || !ok {
		t.Errorf("present: ok=%v err=%v", ok, err)
	}
}

func TestLoad_Missing(t *testing.T) {
	store := New(t.TempDir())
	if _, err := store.Load(context.Background(), "nope"); err == nil {
		t.Error("missing load should error")
	}
}

func ids(apps []domain.Application) []string {
	out := make([]string, len(apps))
	for i, a := range apps {
		out[i] = a.ID
	}
	return out
}
