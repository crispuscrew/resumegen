package domain

import "fmt"

// Status is an application's position in the pipeline. The forward chain
// (drafting->applied->screen->interview->offer->accepted) advances one step at a
// time; the four terminal states have no outgoing transitions.
type Status string

const (
	StatusDrafting  Status = "drafting"
	StatusApplied   Status = "applied"
	StatusScreen    Status = "screen"
	StatusInterview Status = "interview"
	StatusOffer     Status = "offer"
	StatusAccepted  Status = "accepted"  // terminal
	StatusRejected  Status = "rejected"  // terminal
	StatusWithdrawn Status = "withdrawn" // terminal
	StatusGhosted   Status = "ghosted"   // terminal
)

// forwardNext maps each state in the linear chain to the single state it may
// advance to. States absent here (terminal states) have no forward step.
var forwardNext = map[Status]Status{
	StatusDrafting:  StatusApplied,
	StatusApplied:   StatusScreen,
	StatusScreen:    StatusInterview,
	StatusInterview: StatusOffer,
	StatusOffer:     StatusAccepted,
}

// submitted reports whether s is a post-submission active state - the set from
// which `rejected` is reachable (applied and beyond, never drafting).
func (s Status) submitted() bool {
	switch s {
	case StatusApplied, StatusScreen, StatusInterview, StatusOffer:
		return true
	default:
		return false
	}
}

// Terminal reports whether s has no outgoing transitions.
func (s Status) Terminal() bool {
	switch s {
	case StatusAccepted, StatusRejected, StatusWithdrawn, StatusGhosted:
		return true
	default:
		return false
	}
}

// Active reports whether s still admits transitions (the negation of Terminal).
func (s Status) Active() bool { return !s.Terminal() }

// Valid reports whether s is one of the nine known statuses.
func (s Status) Valid() bool {
	switch s {
	case StatusDrafting, StatusApplied, StatusScreen, StatusInterview, StatusOffer,
		StatusAccepted, StatusRejected, StatusWithdrawn, StatusGhosted:
		return true
	default:
		return false
	}
}

// InvalidTransitionError names a rejected (from,to) pair. Callers map it to
// exit code 1 (resolution error).
type InvalidTransitionError struct {
	From, To Status
}

func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("cannot transition from %q to %q", string(e.From), string(e.To))
}

// InvalidStatusError names an unknown status string. Callers map it to exit
// code 2 (usage error).
type InvalidStatusError struct{ Value string }

func (e *InvalidStatusError) Error() string {
	return fmt.Sprintf("unknown status %q", e.Value)
}

// CanTransition returns nil if moving from->to is permitted, else a named error.
// It is pure: it appends no events and never reads the clock - the use case
// does that around it.
//
// Rules (DESIGN section 7.2):
//   - forward chain advances exactly one step (no skipping);
//   - rejected only from a submitted state (applied and beyond), not drafting;
//   - withdrawn from any active state (user-driven);
//   - ghosted from any active state incl. drafting (auto only, section 4);
//   - terminal states have no outgoing transitions.
func CanTransition(from, to Status) error {
	if !from.Valid() {
		return &InvalidStatusError{Value: string(from)}
	}
	if !to.Valid() {
		return &InvalidStatusError{Value: string(to)}
	}
	if from.Terminal() {
		return &InvalidTransitionError{From: from, To: to}
	}
	switch to {
	case StatusWithdrawn:
		// Any active state may be withdrawn.
		return nil
	case StatusGhosted:
		// Any active state may be ghosted (auto path enforces the timing).
		return nil
	case StatusRejected:
		if from.submitted() {
			return nil
		}
		return &InvalidTransitionError{From: from, To: to}
	default:
		if forwardNext[from] == to {
			return nil
		}
		return &InvalidTransitionError{From: from, To: to}
	}
}
