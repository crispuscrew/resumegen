package track

import (
	"fmt"
	"strings"
	"time"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// buildID renders the stable identity <YYYY-MM-DD>_<company-slug>_<role-slug>,
// where the date is the entry's creation day (SPEC §2).
// The date comes from domain.DayFloor so the ID lands on the caller's calendar
// day; formatting now.UTC() directly would re-introduce the off-by-one that
// DayFloor exists to prevent.
func buildID(now time.Time, company, role string) string {
	return fmt.Sprintf("%s_%s_%s", domain.DayFloor(now).Format("2006-01-02"), slug(company), slug(role))
}

// slug lowercases s, maps every non-alphanumeric run to a single '-', and trims
// leading/trailing '-'. An all-symbol input yields "x" so IDs never collapse to
// an empty segment.
func slug(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "x"
	}
	return out
}

// dayOf truncates t to UTC midnight; delegates to the single domain helper.
func dayOf(t time.Time) time.Time { return domain.DayFloor(t) }

// applyTransition sets the new status and appends the status-change event.
func applyTransition(app *domain.Application, to domain.Status, now time.Time, note string) {
	app.Status = to
	app.Events = append(app.Events, domain.Event{At: now, Kind: "status", Note: statusNote(to, note)})
	if to == domain.StatusApplied {
		app.AppliedAt = dayOf(now)
	}
}

// statusNote records the target status in the event note, keeping any operator
// note the caller supplied.
func statusNote(to domain.Status, note string) string {
	if note == "" {
		return string(to)
	}
	return string(to) + ": " + note
}

// advance single-steps app up the forward chain to target, validating each hop
// via domain.CanTransition. Used by New for the --status shortcut.
func advance(app *domain.Application, target domain.Status, now time.Time, note string) error {
	// Bound the loop by the number of known states so a bad target can never
	// spin forever.
	for i := 0; i < 16; i++ {
		if app.Status == target {
			return nil
		}
		next := nextToward(app.Status, target)
		if next == "" {
			return &domain.InvalidTransitionError{From: app.Status, To: target}
		}
		if err := domain.CanTransition(app.Status, next); err != nil {
			return err
		}
		stepNote := ""
		if next == target {
			stepNote = note // the operator's note belongs to the final hop only
		}
		applyTransition(app, next, now, stepNote)
	}
	return &domain.InvalidTransitionError{From: app.Status, To: target}
}

// nextToward returns the next single step from `from` heading toward `target`.
// Terminal targets (rejected/withdrawn/ghosted) are reached directly; a
// forward-chain target advances one link. Returns "" when unreachable.
func nextToward(from, target domain.Status) domain.Status {
	switch target {
	case domain.StatusRejected, domain.StatusWithdrawn, domain.StatusGhosted:
		return target
	}
	chain := []domain.Status{
		domain.StatusDrafting, domain.StatusApplied, domain.StatusScreen,
		domain.StatusInterview, domain.StatusOffer, domain.StatusAccepted,
	}
	fi, ti := indexOf(chain, from), indexOf(chain, target)
	if fi < 0 || ti < 0 || ti <= fi {
		return ""
	}
	return chain[fi+1]
}

func indexOf(chain []domain.Status, s domain.Status) int {
	for i, c := range chain {
		if c == s {
			return i
		}
	}
	return -1
}

// matches reports whether a passes every enabled filter in q (filters AND).
// Staleness and followup-due reuse the pure domain helpers so the filter and the
// list --json contract can never diverge.
func (q Query) matches(a domain.Application, now time.Time) bool {
	if len(q.Statuses) > 0 && !statusIn(a.Status, q.Statuses) {
		return false
	}
	if q.hasStaleDays && !isStale(a, now, q.StaleDays) {
		return false
	}
	if q.FollowupsDue && !a.HasDueFollowup(now) {
		return false
	}
	return true
}

func statusIn(s domain.Status, set []domain.Status) bool {
	for _, x := range set {
		if s == x {
			return true
		}
	}
	return false
}

// isStale reports whether a has had no activity within the last n days: strictly
// more than n days since LastActivity as of now.
func isStale(a domain.Application, now time.Time, n int) bool {
	return now.After(a.LastActivity().Add(time.Duration(n) * 24 * time.Hour))
}
