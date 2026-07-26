package track

import (
	"testing"
	"time"

	"github.com/crispuscrew/resumegen/internal/domain"
)

func statusList(ss []domain.Status) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ","
		}
		out += string(s)
	}
	return out
}

func TestManualTargets(t *testing.T) {
	cases := map[domain.Status]string{
		domain.StatusDrafting:  "applied,withdrawn",
		domain.StatusApplied:   "screen,rejected,withdrawn",
		domain.StatusScreen:    "interview,rejected,withdrawn",
		domain.StatusInterview: "offer,rejected,withdrawn",
		domain.StatusOffer:     "accepted,rejected,withdrawn",
		// terminal states have no outgoing transitions
		domain.StatusAccepted:  "",
		domain.StatusRejected:  "",
		domain.StatusWithdrawn: "",
		domain.StatusGhosted:   "",
	}
	for from, want := range cases {
		if got := statusList(ManualTargets(from)); got != want {
			t.Errorf("ManualTargets(%s) = %q, want %q", from, got, want)
		}
	}
}

func TestManualTargets_NeverOffersGhosted(t *testing.T) {
	// ghosted is auto-only; it must never appear as a manual target from any state.
	for _, from := range StatusesInPipelineOrder() {
		for _, to := range ManualTargets(from) {
			if to == domain.StatusGhosted {
				t.Errorf("ManualTargets(%s) offered ghosted", from)
			}
		}
	}
}

func TestSummarize(t *testing.T) {
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	old := now.Add(-40 * 24 * time.Hour)
	apps := []domain.Application{
		// active, fresh, with a due followup
		{Status: domain.StatusApplied, AppliedAt: now, Events: []domain.Event{{At: now}},
			Followups: []domain.Followup{{Due: now.Add(-24 * time.Hour), Action: "ping"}}},
		// active, stale (last activity 40 days ago)
		{Status: domain.StatusScreen, AppliedAt: old, Events: []domain.Event{{At: old}}},
		// terminal
		{Status: domain.StatusRejected, AppliedAt: old},
		// terminal
		{Status: domain.StatusAccepted, AppliedAt: now},
	}
	s := Summarize(apps, now, 30)
	if s.Total != 4 {
		t.Errorf("Total = %d, want 4", s.Total)
	}
	if s.Active != 2 {
		t.Errorf("Active = %d, want 2", s.Active)
	}
	if s.Terminal != 2 {
		t.Errorf("Terminal = %d, want 2", s.Terminal)
	}
	if s.StaleActive != 1 {
		t.Errorf("StaleActive = %d, want 1 (the 40-day-old screen)", s.StaleActive)
	}
	if s.FollowupsDue != 1 {
		t.Errorf("FollowupsDue = %d, want 1", s.FollowupsDue)
	}
	if s.ByStatus[domain.StatusApplied] != 1 || s.ByStatus[domain.StatusScreen] != 1 ||
		s.ByStatus[domain.StatusRejected] != 1 || s.ByStatus[domain.StatusAccepted] != 1 {
		t.Errorf("ByStatus wrong: %v", s.ByStatus)
	}
}

func TestStaleWarnDays(t *testing.T) {
	cases := map[int]int{
		30: 22, // 3/4, truncated
		20: 15,
		4:  3,
		1:  1, // floor
		0:  1, // degenerate input still warns at 1 day
	}
	for ghost, want := range cases {
		if got := StaleWarnDays(ghost); got != want {
			t.Errorf("StaleWarnDays(%d) = %d, want %d", ghost, got, want)
		}
	}
	// The invariant that makes the dashboard badge live: warn < ghost for any
	// realistic threshold (>= 2 days).
	for ghost := 2; ghost <= 365; ghost++ {
		if StaleWarnDays(ghost) >= ghost {
			t.Fatalf("StaleWarnDays(%d) = %d, not below the ghost threshold", ghost, StaleWarnDays(ghost))
		}
	}
}

func TestSummarize_StaleDisabled(t *testing.T) {
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	old := now.Add(-400 * 24 * time.Hour)
	apps := []domain.Application{{Status: domain.StatusApplied, AppliedAt: old, Events: []domain.Event{{At: old}}}}
	if s := Summarize(apps, now, 0); s.StaleActive != 0 {
		t.Errorf("staleAfterDays=0 should disable stale count, got %d", s.StaleActive)
	}
}
