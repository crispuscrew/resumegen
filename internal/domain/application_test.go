package domain

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestLastActivityAndStaleDays(t *testing.T) {
	base := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
	a := Application{
		AppliedAt: base,
		Events: []Event{
			{At: base, Kind: "created"},
			{At: base.Add(48 * time.Hour), Kind: "note"},
		},
	}
	if !a.LastActivity().Equal(base.Add(48 * time.Hour)) {
		t.Errorf("LastActivity = %v, want latest event", a.LastActivity())
	}
	now := base.Add(48*time.Hour + 5*24*time.Hour) // 5 days after last activity
	if got := a.StaleDays(now); got != 5 {
		t.Errorf("StaleDays = %d, want 5", got)
	}
	if got := a.StaleDays(base); got != 0 { // now before last activity → clamped to 0
		t.Errorf("StaleDays negative guard = %d, want 0", got)
	}
}

func TestHasDueFollowup(t *testing.T) {
	today := time.Date(2026, time.July, 10, 8, 0, 0, 0, time.UTC)
	a := Application{Followups: []Followup{
		{Due: today.Add(-24 * time.Hour), Action: "overdue"},
		{Due: today.Add(72 * time.Hour), Action: "future"},
	}}
	if !a.HasDueFollowup(today) {
		t.Error("an overdue undone followup should be due")
	}
	a.Followups[0].Done = true
	if a.HasDueFollowup(today) {
		t.Error("only a future followup remains; nothing due")
	}
	b := Application{Followups: []Followup{{Due: today, Action: "today"}}}
	if !b.HasDueFollowup(today) {
		t.Error("a followup due today should count")
	}
}

func TestGhostDue(t *testing.T) {
	now := day(2026, time.July, 10)
	cases := []struct {
		name string
		app  Application
		days int
		want bool
	}{
		{
			name: "active and stale by events",
			app:  Application{Status: StatusApplied, AppliedAt: day(2026, time.May, 1), Events: []Event{{At: day(2026, time.May, 1)}}},
			days: 30, want: true,
		},
		{
			name: "active but fresh",
			app:  Application{Status: StatusApplied, AppliedAt: day(2026, time.July, 5), Events: []Event{{At: day(2026, time.July, 5)}}},
			days: 30, want: false,
		},
		{
			name: "terminal never ghosts",
			app:  Application{Status: StatusRejected, AppliedAt: day(2026, time.January, 1)},
			days: 30, want: false,
		},
		{
			name: "drafting never ghosts (was never submitted)",
			app:  Application{Status: StatusDrafting, AppliedAt: day(2026, time.January, 1), Events: []Event{{At: day(2026, time.January, 1)}}},
			days: 30, want: false,
		},
		{
			name: "uses latest event not applied_at",
			app:  Application{Status: StatusScreen, AppliedAt: day(2026, time.January, 1), Events: []Event{{At: day(2026, time.January, 1)}, {At: day(2026, time.July, 8)}}},
			days: 30, want: false,
		},
		{
			name: "no events falls back to applied_at",
			app:  Application{Status: StatusApplied, AppliedAt: day(2026, time.May, 1)},
			days: 30, want: true,
		},
		{
			name: "zero threshold disables",
			app:  Application{Status: StatusApplied, AppliedAt: day(2020, time.January, 1)},
			days: 0, want: false,
		},
		{
			name: "exactly at boundary is not yet due",
			app:  Application{Status: StatusApplied, AppliedAt: day(2026, time.June, 10), Events: []Event{{At: day(2026, time.June, 10)}}},
			days: 30, want: false, // June 10 + 30d = July 10 == now, not After
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GhostDue(c.app, now, c.days); got != c.want {
				t.Errorf("GhostDue = %v, want %v", got, c.want)
			}
		})
	}
}
