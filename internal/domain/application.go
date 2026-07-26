package domain

import "time"

// Application is one tracked job application: where you applied, what artifacts
// went out, the current pipeline status, and an append-only event log. It is a
// pure value — `time.Time` is permitted (archtest bans only IO stdlib), but the
// use case injects `now`; domain never reads the clock.
//
// The struct `toml:"..."` tags are metadata only; they do not import go-toml.
// The trackrepo adapter owns marshal/unmarshal and the local-date/offset-
// datetime mapping (DESIGN §7.1), mirroring how config.go carries tags but
// tomlrepo does the loading.
type Application struct {
	ID          string       `toml:"id"`
	Company     string       `toml:"company"`
	Role        string       `toml:"role"`
	Source      string       `toml:"source"`       // where the JD came from (free text: "hh.ru", "referral")
	Profile     string       `toml:"profile"`      // resume profile used
	JDPath      string       `toml:"jd_path"`      // optional JD text file you keep; never fetched/parsed
	ResumePDF   string       `toml:"resume_pdf"`   //
	CoverLetter string       `toml:"cover_letter"` //
	Status      Status       `toml:"status"`
	SalaryRange string       `toml:"salary_range"`
	Remote      bool         `toml:"remote"`
	AppliedAt   time.Time    `toml:"applied_at"` // day-granularity (see trackrepo §3)
	Notes       string       `toml:"notes"`
	Contacts    []AppContact `toml:"contacts"`
	Followups   []Followup   `toml:"followups"`
	Events      []Event      `toml:"events"` // append-only
}

// Event is one entry in the append-only log. At is stored as an offset-datetime
// by the adapter; Kind is a short verb ("created", "note", "status", "ghosted").
type Event struct {
	At   time.Time `toml:"at"`
	Kind string    `toml:"kind"`
	Note string    `toml:"note"`
}

// Followup is a reminder to act by Due (day-granularity local-date on disk).
type Followup struct {
	Due    time.Time `toml:"due"`
	Action string    `toml:"action"`
	Done   bool      `toml:"done"`
}

// AppContact is a person tied to an application (named to avoid colliding with
// the resume-header Contact). Round-trips hand-edited entries; there is no
// `apply contact` command in v1.4 (SPEC §10).
type AppContact struct {
	Name        string    `toml:"name"`
	Role        string    `toml:"role"`
	Channel     string    `toml:"channel"`
	LastContact time.Time `toml:"last_contact"`
}

// LastActivity returns the timestamp of the most recent event, or AppliedAt if
// there are no events. Used to measure staleness (GhostDue, StaleDays).
func (a Application) LastActivity() time.Time {
	last := a.AppliedAt
	for _, e := range a.Events {
		if e.At.After(last) {
			last = e.At
		}
	}
	return last
}

// StaleDays returns the whole days elapsed since LastActivity as of now, never
// negative. It is a display value (the list --json contract); the stale *filter*
// uses the same LastActivity floor with strict "more than N days" semantics.
func (a Application) StaleDays(now time.Time) int {
	d := now.Sub(a.LastActivity())
	if d < 0 {
		return 0
	}
	return int(d / (24 * time.Hour))
}

// HasDueFollowup reports whether a has an undone followup due on or before the
// day of now. Pure; shared by the query filter and the list --json contract.
func (a Application) HasDueFollowup(now time.Time) bool {
	today := DayFloor(now)
	for _, f := range a.Followups {
		if f.Done {
			continue
		}
		if !DayFloor(f.Due).After(today) {
			return true
		}
	}
	return false
}

// NextFollowupDue returns the earliest due date among undone followups, or the
// zero time when none are pending. Pure; shared by the list table and --json.
func (a Application) NextFollowupDue() time.Time {
	var next time.Time
	for _, f := range a.Followups {
		if f.Done {
			continue
		}
		if next.IsZero() || f.Due.Before(next) {
			next = f.Due
		}
	}
	return next
}

// GhostDue reports whether a is a stale active application that should be
// auto-ghosted: it is true iff a.Status is Active and the most recent activity
// (latest event At, or AppliedAt when there are no events) is more than
// ghostAfterDays days before now. It is pure and deterministic given now; the
// use case applies the transition and persists (SPEC §4).
func GhostDue(a Application, now time.Time, ghostAfterDays int) bool {
	// Only submitted applications ghost: "ghosted" means the employer went
	// silent after you applied. A stale draft was never sent, so it just stays
	// a draft (delete or edit it instead).
	if !a.Status.submitted() {
		return false
	}
	if ghostAfterDays <= 0 {
		return false
	}
	deadline := a.LastActivity().Add(time.Duration(ghostAfterDays) * 24 * time.Hour)
	return now.After(deadline)
}

// DayFloor truncates t to the midnight of t's OWN calendar day, represented as
// UTC midnight to match the on-disk local-date storage. The single
// day-flooring implementation for every layer.
//
// The calendar day is read from t in its own location, not from t.UTC(): a
// clock of time.Now() east or west of UTC otherwise yields a "today" that is
// the UTC day rather than the user's, shifting every ID, applied_at, and
// followup due date by one for a window equal to the UTC offset.
func DayFloor(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
