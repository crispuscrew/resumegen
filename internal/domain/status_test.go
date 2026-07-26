package domain

import (
	"errors"
	"testing"
)

func asErr(err error, target any) bool { return errors.As(err, target) }

// allStatuses is every known status, used to drive the all-pairs table.
var allStatuses = []Status{
	StatusDrafting, StatusApplied, StatusScreen, StatusInterview, StatusOffer,
	StatusAccepted, StatusRejected, StatusWithdrawn, StatusGhosted,
}

// allowed is the ground-truth transition set the state machine must implement.
// key "from->to" present means CanTransition(from,to) must be nil.
func allowedEdges() map[string]bool {
	edges := map[string]bool{}
	add := func(from, to Status) { edges[string(from)+"->"+string(to)] = true }

	// forward chain, one step each
	add(StatusDrafting, StatusApplied)
	add(StatusApplied, StatusScreen)
	add(StatusScreen, StatusInterview)
	add(StatusInterview, StatusOffer)
	add(StatusOffer, StatusAccepted)

	// withdrawn: from any active state
	for _, s := range []Status{StatusDrafting, StatusApplied, StatusScreen, StatusInterview, StatusOffer} {
		add(s, StatusWithdrawn)
	}
	// ghosted: from any active state (incl. drafting)
	for _, s := range []Status{StatusDrafting, StatusApplied, StatusScreen, StatusInterview, StatusOffer} {
		add(s, StatusGhosted)
	}
	// rejected: only from a submitted state (applied and beyond)
	for _, s := range []Status{StatusApplied, StatusScreen, StatusInterview, StatusOffer} {
		add(s, StatusRejected)
	}
	return edges
}

func TestCanTransition_AllPairs(t *testing.T) {
	allowed := allowedEdges()
	for _, from := range allStatuses {
		for _, to := range allStatuses {
			key := string(from) + "->" + string(to)
			err := CanTransition(from, to)
			want := allowed[key]
			if want && err != nil {
				t.Errorf("%s: want allowed, got error %v", key, err)
			}
			if !want && err == nil {
				t.Errorf("%s: want rejected, got nil", key)
			}
		}
	}
}

func TestCanTransition_TerminalRejectsEverything(t *testing.T) {
	terminals := []Status{StatusAccepted, StatusRejected, StatusWithdrawn, StatusGhosted}
	for _, from := range terminals {
		for _, to := range allStatuses {
			if err := CanTransition(from, to); err == nil {
				t.Errorf("terminal %s -> %s: want error, got nil", from, to)
			}
		}
	}
}

func TestCanTransition_InvalidStatus(t *testing.T) {
	if err := CanTransition(Status("bogus"), StatusApplied); err == nil {
		t.Error("unknown from status should error")
	}
	if err := CanTransition(StatusDrafting, Status("bogus")); err == nil {
		t.Error("unknown to status should error")
	}
	var ise *InvalidStatusError
	if err := CanTransition(StatusDrafting, Status("bogus")); err == nil {
		t.Fatal("expected error")
	} else if !asErr(err, &ise) {
		t.Errorf("want *InvalidStatusError, got %T", err)
	}
}

func TestCanTransition_NoSkipping(t *testing.T) {
	// drafting cannot jump straight to interview
	if err := CanTransition(StatusDrafting, StatusInterview); err == nil {
		t.Error("drafting -> interview should be rejected (single-step only)")
	}
	var ite *InvalidTransitionError
	if err := CanTransition(StatusDrafting, StatusInterview); !asErr(err, &ite) {
		t.Errorf("want *InvalidTransitionError, got %T", err)
	}
}

func TestCanTransition_RejectedNotFromDrafting(t *testing.T) {
	if err := CanTransition(StatusDrafting, StatusRejected); err == nil {
		t.Error("drafting -> rejected must be rejected (use withdrawn)")
	}
}

func TestStatus_TerminalAndActive(t *testing.T) {
	active := []Status{StatusDrafting, StatusApplied, StatusScreen, StatusInterview, StatusOffer}
	for _, s := range active {
		if s.Terminal() || !s.Active() {
			t.Errorf("%s: want active", s)
		}
	}
	terminal := []Status{StatusAccepted, StatusRejected, StatusWithdrawn, StatusGhosted}
	for _, s := range terminal {
		if !s.Terminal() || s.Active() {
			t.Errorf("%s: want terminal", s)
		}
	}
}

func TestStatus_Valid(t *testing.T) {
	for _, s := range allStatuses {
		if !s.Valid() {
			t.Errorf("%s: want valid", s)
		}
	}
	if Status("nope").Valid() {
		t.Error("unknown status should be invalid")
	}
}
