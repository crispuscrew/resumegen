package cli

import (
	"time"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// Stable JSON shapes for the apply agent contract (--json). These DTOs pin the
// wire format independently of the domain types, so refactors can't silently
// change the contract (same discipline as the v1.3 prompt DTOs).

const dateJSON = "2006-01-02"

type eventJSON struct {
	At   string `json:"at"`
	Kind string `json:"kind"`
	Note string `json:"note,omitempty"`
}

type followupJSON struct {
	Due    string `json:"due"`
	Action string `json:"action"`
	Done   bool   `json:"done"`
}

type contactJSON struct {
	Name        string `json:"name,omitempty"`
	Role        string `json:"role,omitempty"`
	Channel     string `json:"channel,omitempty"`
	LastContact string `json:"last_contact,omitempty"`
}

// applyListItemJSON is one row of `apply list --json` (named distinctly from the
// prompt layer's listItemJSON). stale_days/followups_due are computed from the
// same pure domain helpers the filters use.
type applyListItemJSON struct {
	ID           string `json:"id"`
	Company      string `json:"company"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	AppliedAt    string `json:"applied_at,omitempty"`
	StaleDays    int    `json:"stale_days"`
	FollowupsDue bool   `json:"followups_due"`
	NextDue      string `json:"next_due,omitempty"`
}

func listApplicationsJSON(apps []domain.Application, now time.Time) []applyListItemJSON {
	out := make([]applyListItemJSON, 0, len(apps))
	for _, a := range apps {
		out = append(out, applyListItemJSON{
			ID:           a.ID,
			Company:      a.Company,
			Role:         a.Role,
			Status:       string(a.Status),
			AppliedAt:    dayString(a.AppliedAt),
			StaleDays:    a.StaleDays(now),
			FollowupsDue: a.HasDueFollowup(now),
			NextDue:      dayString(a.NextFollowupDue()),
		})
	}
	return out
}

// applicationJSON is the full `apply show --json` object.
type applicationJSON struct {
	ID          string         `json:"id"`
	Company     string         `json:"company"`
	Role        string         `json:"role"`
	Source      string         `json:"source,omitempty"`
	Profile     string         `json:"profile,omitempty"`
	JDPath      string         `json:"jd_path,omitempty"`
	ResumePDF   string         `json:"resume_pdf,omitempty"`
	CoverLetter string         `json:"cover_letter,omitempty"`
	Status      string         `json:"status"`
	SalaryRange string         `json:"salary_range,omitempty"`
	Remote      bool           `json:"remote"`
	AppliedAt   string         `json:"applied_at,omitempty"`
	Notes       string         `json:"notes,omitempty"`
	Contacts    []contactJSON  `json:"contacts"`
	Followups   []followupJSON `json:"followups"`
	Events      []eventJSON    `json:"events"`
}

func showApplicationJSON(a domain.Application) applicationJSON {
	out := applicationJSON{
		ID:          a.ID,
		Company:     a.Company,
		Role:        a.Role,
		Source:      a.Source,
		Profile:     a.Profile,
		JDPath:      a.JDPath,
		ResumePDF:   a.ResumePDF,
		CoverLetter: a.CoverLetter,
		Status:      string(a.Status),
		SalaryRange: a.SalaryRange,
		Remote:      a.Remote,
		AppliedAt:   dayString(a.AppliedAt),
		Notes:       a.Notes,
		Contacts:    make([]contactJSON, 0, len(a.Contacts)),
		Followups:   make([]followupJSON, 0, len(a.Followups)),
		Events:      make([]eventJSON, 0, len(a.Events)),
	}
	for _, c := range a.Contacts {
		out.Contacts = append(out.Contacts, contactJSON{
			Name: c.Name, Role: c.Role, Channel: c.Channel,
			LastContact: dayString(c.LastContact),
		})
	}
	for _, f := range a.Followups {
		out.Followups = append(out.Followups, followupJSON{
			Due: dayString(f.Due), Action: f.Action, Done: f.Done,
		})
	}
	for _, e := range a.Events {
		out.Events = append(out.Events, eventJSON{
			At: e.At.UTC().Format(time.RFC3339), Kind: e.Kind, Note: e.Note,
		})
	}
	return out
}

// dayString formats a day-granularity time as YYYY-MM-DD, or "" when zero.
func dayString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(dateJSON)
}
