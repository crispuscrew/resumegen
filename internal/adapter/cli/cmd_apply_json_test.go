package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// These tests PIN the apply --json wire format (the agent contract). If a
// marshal here changes shape, that is a breaking change to scripted callers —
// think before touching the expectations.

func contractApp() domain.Application {
	at := time.Date(2026, time.July, 1, 10, 0, 0, 0, time.UTC)
	return domain.Application{
		ID: "2026-07-01_acme_go", Company: "Acme", Role: "Go Dev",
		Source: "hh.ru", Profile: "go-backend", JDPath: "/jd/acme.md",
		Status: domain.StatusScreen, SalaryRange: "300-400k", Remote: true,
		AppliedAt: at, Notes: "warm intro",
		Contacts: []domain.AppContact{{Name: "Ivan", Role: "recruiter", Channel: "tg", LastContact: at}},
		Followups: []domain.Followup{
			{Due: at.AddDate(0, 0, 7), Action: "nudge", Done: false},
			{Due: at.AddDate(0, 0, 2), Action: "thank", Done: true},
		},
		Events: []domain.Event{{At: at, Kind: "created"}},
	}
}

func TestApplyJSON_ShowContract(t *testing.T) {
	raw, err := json.Marshal(showApplicationJSON(contractApp()))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"id": "2026-07-01_acme_go", "company": "Acme", "role": "Go Dev",
		"source": "hh.ru", "profile": "go-backend", "jd_path": "/jd/acme.md",
		"status": "screen", "salary_range": "300-400k", "remote": true,
		"applied_at": "2026-07-01", "notes": "warm intro",
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("show json %q = %v, want %v", k, m[k], v)
		}
	}
	for _, k := range []string{"contacts", "followups", "events"} {
		if _, ok := m[k]; !ok {
			t.Errorf("show json missing %q", k)
		}
	}
}

func TestApplyJSON_ListContract(t *testing.T) {
	now := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	rows := listApplicationsJSON([]domain.Application{contractApp()}, now)
	raw, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("rows = %d", len(out))
	}
	m := out[0]
	want := map[string]any{
		"id": "2026-07-01_acme_go", "company": "Acme", "role": "Go Dev",
		"status": "screen", "applied_at": "2026-07-01",
		"stale_days": float64(9), "followups_due": true,
		// next_due = earliest UNDONE followup (the done one is skipped)
		"next_due": "2026-07-08",
	}
	for k, v := range want {
		if m[k] != v {
			t.Errorf("list json %q = %v, want %v", k, m[k], v)
		}
	}
}

func TestApplyJSON_ZeroAppliedAtOmitted(t *testing.T) {
	a := contractApp()
	a.AppliedAt = time.Time{} // drafting: never applied
	raw, _ := json.Marshal(listApplicationsJSON([]domain.Application{a}, time.Date(2026, time.July, 2, 0, 0, 0, 0, time.UTC)))
	var out []map[string]any
	_ = json.Unmarshal(raw, &out)
	if _, present := out[0]["applied_at"]; present {
		t.Error("zero applied_at must be omitted, not \"0001-01-01\"")
	}
}
