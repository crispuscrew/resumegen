package track

import (
	"time"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// statusMenuOrder is the canonical ordering for status choices a user may pick
// in a UI. It omits drafting (never a transition target) and ghosted (applied
// automatically, never by hand — see Transition).
var statusMenuOrder = []domain.Status{
	domain.StatusApplied,
	domain.StatusScreen,
	domain.StatusInterview,
	domain.StatusOffer,
	domain.StatusAccepted,
	domain.StatusRejected,
	domain.StatusWithdrawn,
}

// ManualTargets returns the statuses a user may transition `from` into by hand,
// in menu order. It mirrors Transition's policy exactly: the pure state machine
// must permit the edge (domain.CanTransition) and, because ghosted is excluded
// from statusMenuOrder, the auto-only ghosted target is never offered. A
// terminal `from` yields an empty slice (no outgoing transitions).
func ManualTargets(from domain.Status) []domain.Status {
	out := make([]domain.Status, 0, len(statusMenuOrder))
	for _, to := range statusMenuOrder {
		if domain.CanTransition(from, to) == nil {
			out = append(out, to)
		}
	}
	return out
}

// Summary is an aggregate view of a set of applications, for a dashboard.
type Summary struct {
	Total        int
	Active       int
	Terminal     int
	StaleActive  int // active entries with no activity in >= staleAfterDays days
	FollowupsDue int // entries with an undone followup due on/before today
	ByStatus     map[domain.Status]int
}

// Summarize aggregates apps as of now. staleAfterDays <= 0 disables the stale
// count (leaves StaleActive at 0). It reads only pure domain methods, so it is
// deterministic given a fixed now — the TUI renders the result and computes
// nothing itself.
func Summarize(apps []domain.Application, now time.Time, staleAfterDays int) Summary {
	s := Summary{Total: len(apps), ByStatus: make(map[domain.Status]int)}
	for _, a := range apps {
		s.ByStatus[a.Status]++
		if a.Status.Active() {
			s.Active++
			if staleAfterDays > 0 && a.StaleDays(now) >= staleAfterDays {
				s.StaleActive++
			}
		} else {
			s.Terminal++
		}
		if a.HasDueFollowup(now) {
			s.FollowupsDue++
		}
	}
	return s
}

// StaleWarnDays returns the inactivity threshold at which a dashboard should
// WARN about an active application, given the auto-ghost threshold. It is 3/4
// of ghostAfterDays (min 1 day): warning at the ghost threshold itself is dead
// by construction, because List auto-ghosts everything at/past that mark before
// any summary runs — the warning must fire earlier to be actionable.
func StaleWarnDays(ghostAfterDays int) int {
	warn := ghostAfterDays * 3 / 4
	if warn < 1 {
		warn = 1
	}
	return warn
}

// StatusesInPipelineOrder returns all nine statuses in pipeline order, for
// stable dashboard rendering. Callers skip zero counts as they see fit.
func StatusesInPipelineOrder() []domain.Status {
	return []domain.Status{
		domain.StatusDrafting, domain.StatusApplied, domain.StatusScreen,
		domain.StatusInterview, domain.StatusOffer, domain.StatusAccepted,
		domain.StatusRejected, domain.StatusWithdrawn, domain.StatusGhosted,
	}
}
