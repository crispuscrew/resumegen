// Package track is the application-tracker use case: it builds application IDs,
// enforces the pure status state machine, appends events, and applies lazy
// auto-ghosting on read. It performs no IO and calls no LLM - persistence lives
// behind the Store port (implemented by adapter/trackrepo) and `now` is injected
// so the ghost/stale/date surface is deterministic under test.
package track

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// Store persists applications, one <id>.toml file each. Implemented by
// adapter/trackrepo; the id is already assigned before Save.
type Store interface {
	// Save marshals and writes the application (its ID is authoritative).
	Save(ctx context.Context, a domain.Application) error
	// Load reads and unmarshals the application with the given id.
	Load(ctx context.Context, id string) (domain.Application, error)
	// List reads every applications/*.toml entry.
	List(ctx context.Context) ([]domain.Application, error)
	// Exists reports whether an entry with id is already stored (for
	// collision-suffixing in New).
	Exists(ctx context.Context, id string) (bool, error)
	// Delete removes the entry with the given id.
	Delete(ctx context.Context, id string) error
}

// Tracker orchestrates the Store. Now is injected (time.Now in cmd, a fixed
// clock in tests). GhostAfterDays comes from config (default applied by cmd).
type Tracker struct {
	Store           Store
	Now             func() time.Time
	GhostAfterDays  int
	FollowupLagDays int // default due-date lag when AddFollowup gets a zero due
}

// NewInput is the payload for creating an application.
type NewInput struct {
	Company     string
	Role        string
	Profile     string
	Source      string
	JDPath      string
	SalaryRange string
	Remote      bool
	Status      domain.Status // optional; when set and past drafting, New single-steps up to it
}

// Query filters List results. Zero value matches everything.
type Query struct {
	Statuses     []domain.Status // OR within the set; empty matches all
	StaleDays    int             // >0: keep entries with no activity in N days
	FollowupsDue bool            // keep entries with an undone followup due on/before today
	hasStaleDays bool            // set by the CLI to distinguish 0 from unset
}

// WithStaleDays returns a copy of q with the stale filter enabled at n days.
// n<=0 disables the filter (matches everything on that axis).
func (q Query) WithStaleDays(n int) Query {
	q.StaleDays = n
	q.hasStaleDays = n > 0
	return q
}

func (t *Tracker) now() time.Time { return t.Now() }

// New builds a drafting application: it slugs an ID from today's date +
// company + role, resolves collisions via Store.Exists, appends a "created"
// event, and Saves. When in.Status is set beyond drafting it single-steps the
// entry up the chain in-process (each step validated), so recording an
// already-submitted application is one call.
func (t *Tracker) New(ctx context.Context, in NewInput) (domain.Application, error) {
	if strings.TrimSpace(in.Company) == "" || strings.TrimSpace(in.Role) == "" {
		return domain.Application{}, fmt.Errorf("company and role are required")
	}
	now := t.now()
	id, err := t.assignID(ctx, now, in.Company, in.Role)
	if err != nil {
		return domain.Application{}, err
	}
	app := domain.Application{
		ID:          id,
		Company:     in.Company,
		Role:        in.Role,
		Profile:     in.Profile,
		Source:      in.Source,
		JDPath:      in.JDPath,
		SalaryRange: in.SalaryRange,
		Remote:      in.Remote,
		Status:      domain.StatusDrafting,
		// AppliedAt stays zero until the entry actually reaches "applied" -
		// a drafting entry has not been applied, so no date is shown for it.
		Events: []domain.Event{
			{At: now, Kind: "created", Note: ""},
		},
	}

	if target := in.Status; target != "" && target != domain.StatusDrafting {
		if !target.Valid() {
			return domain.Application{}, &domain.InvalidStatusError{Value: string(target)}
		}
		if target == domain.StatusGhosted {
			return domain.Application{}, fmt.Errorf("cannot create an application as %q: ghosted is applied automatically", domain.StatusGhosted)
		}
		if err := advance(&app, target, now, ""); err != nil {
			return domain.Application{}, err
		}
	}

	if err := t.Store.Save(ctx, app); err != nil {
		return domain.Application{}, err
	}
	return app, nil
}

// Get loads one application and applies lazy auto-ghost (persisting if the
// entry changed). Deterministic given the injected now.
func (t *Tracker) Get(ctx context.Context, id string) (domain.Application, error) {
	app, err := t.Store.Load(ctx, id)
	if err != nil {
		return domain.Application{}, err
	}
	return t.ghostIfDue(ctx, app)
}

// List loads every application, applies lazy auto-ghost to each, then filters
// per q. Results are sorted by ID for deterministic output.
func (t *Tracker) List(ctx context.Context, q Query) ([]domain.Application, error) {
	apps, err := t.Store.List(ctx)
	if err != nil {
		return nil, err
	}
	now := t.now()
	out := make([]domain.Application, 0, len(apps))
	for _, a := range apps {
		a, err = t.ghostIfDue(ctx, a)
		if err != nil {
			return nil, err
		}
		if q.matches(a, now) {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Transition is the only mutator of Status: it validates via
// domain.CanTransition, appends a "status" event stamped at now, and Saves.
func (t *Tracker) Transition(ctx context.Context, id string, to domain.Status, note string) (domain.Application, error) {
	app, err := t.Get(ctx, id)
	if err != nil {
		return domain.Application{}, err
	}
	if to == domain.StatusGhosted {
		// ghosted is auto-only (SPEC section 1.1): the state machine permits the edge for
		// the lazy-ghost path, but a user cannot set it by hand.
		return domain.Application{}, fmt.Errorf("ghosted is applied automatically after inactivity; use %q to close an application yourself", domain.StatusWithdrawn)
	}
	now := t.now()
	if err := domain.CanTransition(app.Status, to); err != nil {
		return domain.Application{}, err
	}
	applyTransition(&app, to, now, note)
	if err := t.Store.Save(ctx, app); err != nil {
		return domain.Application{}, err
	}
	return app, nil
}

// AddFollowup appends a followup and a "followup" event, then Saves.
func (t *Tracker) AddFollowup(ctx context.Context, id string, due time.Time, action string) (domain.Application, error) {
	if strings.TrimSpace(action) == "" {
		return domain.Application{}, fmt.Errorf("followup action is required")
	}
	app, err := t.Get(ctx, id)
	if err != nil {
		return domain.Application{}, err
	}
	now := t.now()
	if due.IsZero() {
		// The "blank due = today + configured lag" rule lives here, once, so the
		// CLI and TUI can never compute different dates for the same input.
		due = now.AddDate(0, 0, t.FollowupLagDays)
	}
	app.Followups = append(app.Followups, domain.Followup{Due: dayOf(due), Action: action})
	app.Events = append(app.Events, domain.Event{At: now, Kind: "followup", Note: action})
	if err := t.Store.Save(ctx, app); err != nil {
		return domain.Application{}, err
	}
	return app, nil
}

// AddNote appends a "note" event. For a drafting entry it also edits Notes in
// place (a drafting application is still being composed).
func (t *Tracker) AddNote(ctx context.Context, id, text string) (domain.Application, error) {
	if strings.TrimSpace(text) == "" {
		return domain.Application{}, fmt.Errorf("note text is required")
	}
	app, err := t.Get(ctx, id)
	if err != nil {
		return domain.Application{}, err
	}
	now := t.now()
	app.Events = append(app.Events, domain.Event{At: now, Kind: "note", Note: text})
	if app.Status == domain.StatusDrafting {
		if app.Notes == "" {
			app.Notes = text
		} else {
			app.Notes = app.Notes + "\n" + text
		}
	}
	if err := t.Store.Save(ctx, app); err != nil {
		return domain.Application{}, err
	}
	return app, nil
}

// ghostIfDue applies the pure GhostDue rule: on a stale active entry it
// transitions to ghosted, appends one "ghosted" event at now, and persists the
// mutated file. Idempotent - a terminal entry is returned untouched.
func (t *Tracker) ghostIfDue(ctx context.Context, app domain.Application) (domain.Application, error) {
	if !domain.GhostDue(app, t.now(), t.GhostAfterDays) {
		return app, nil
	}
	now := t.now()
	applyTransition(&app, domain.StatusGhosted, now, "auto-ghosted: no activity")
	if err := t.Store.Save(ctx, app); err != nil {
		return domain.Application{}, err
	}
	return app, nil
}

// assignID builds <date>_<company>_<role> and appends the first free -N suffix
// when the base id is already taken.
func (t *Tracker) assignID(ctx context.Context, now time.Time, company, role string) (string, error) {
	base := buildID(now, company, role)
	id := base
	for n := 2; ; n++ {
		exists, err := t.Store.Exists(ctx, id)
		if err != nil {
			return "", err
		}
		if !exists {
			return id, nil
		}
		id = fmt.Sprintf("%s-%d", base, n)
	}
}

// Set moves an application to target, auto-advancing through the forward chain
// when target is more than one step ahead (each hop validated and evented), so
// "recruiter called me straight to interview" is one command. Terminal targets
// (rejected/withdrawn) go direct; manual ghosted stays blocked.
func (t *Tracker) Set(ctx context.Context, id string, to domain.Status, note string) (domain.Application, error) {
	if to == domain.StatusGhosted {
		return domain.Application{}, fmt.Errorf("ghosted is applied automatically after inactivity; use %q to close an application yourself", domain.StatusWithdrawn)
	}
	app, err := t.Get(ctx, id)
	if err != nil {
		return domain.Application{}, err
	}
	if app.Status == to {
		// "No transition is silent": a self-set is a typo, not a no-op success.
		return domain.Application{}, fmt.Errorf("%s is already %s", id, to)
	}
	if err := advance(&app, to, t.now(), note); err != nil {
		return domain.Application{}, err
	}
	if err := t.Store.Save(ctx, app); err != nil {
		return domain.Application{}, err
	}
	return app, nil
}

// CompleteFollowup marks the index-th followup (1-based, as displayed) done and
// appends a "followup-done" event.
func (t *Tracker) CompleteFollowup(ctx context.Context, id string, index int) (domain.Application, error) {
	app, err := t.Get(ctx, id)
	if err != nil {
		return domain.Application{}, err
	}
	if index < 1 || index > len(app.Followups) {
		return domain.Application{}, fmt.Errorf("followup %d does not exist (%s has %d)", index, id, len(app.Followups))
	}
	f := &app.Followups[index-1]
	if f.Done {
		return domain.Application{}, fmt.Errorf("followup %d (%q) is already done", index, f.Action)
	}
	f.Done = true
	app.Events = append(app.Events, domain.Event{At: t.now(), Kind: "followup-done", Note: f.Action})
	if err := t.Store.Save(ctx, app); err != nil {
		return domain.Application{}, err
	}
	return app, nil
}

// EditInput carries optional field updates: nil leaves a field untouched.
type EditInput struct {
	Company     *string
	Role        *string
	Profile     *string
	Source      *string
	JDPath      *string
	SalaryRange *string
	ResumePDF   *string
	CoverLetter *string
	Notes       *string
	Remote      *bool
}

// Edit updates the provided fields in place and appends an "edited" event
// naming what changed. The ID is stable identity and is never renamed, even
// when company/role change. Company and role cannot be cleared.
func (t *Tracker) Edit(ctx context.Context, id string, in EditInput) (domain.Application, error) {
	app, err := t.Get(ctx, id)
	if err != nil {
		return domain.Application{}, err
	}
	var changed []string
	setStr := func(name string, dst *string, v *string, required bool) error {
		if v == nil {
			return nil
		}
		nv := strings.TrimSpace(*v)
		if required && nv == "" {
			return fmt.Errorf("%s cannot be empty", name)
		}
		if *dst != nv {
			*dst = nv
			changed = append(changed, name)
		}
		return nil
	}
	if err := setStr("company", &app.Company, in.Company, true); err != nil {
		return domain.Application{}, err
	}
	if err := setStr("role", &app.Role, in.Role, true); err != nil {
		return domain.Application{}, err
	}
	_ = setStr("profile", &app.Profile, in.Profile, false)
	_ = setStr("source", &app.Source, in.Source, false)
	_ = setStr("jd_path", &app.JDPath, in.JDPath, false)
	_ = setStr("salary_range", &app.SalaryRange, in.SalaryRange, false)
	_ = setStr("resume_pdf", &app.ResumePDF, in.ResumePDF, false)
	_ = setStr("cover_letter", &app.CoverLetter, in.CoverLetter, false)
	_ = setStr("notes", &app.Notes, in.Notes, false)
	if in.Remote != nil && app.Remote != *in.Remote {
		app.Remote = *in.Remote
		changed = append(changed, "remote")
	}
	if len(changed) == 0 {
		return domain.Application{}, fmt.Errorf("nothing to edit: no field changed")
	}
	app.Events = append(app.Events, domain.Event{At: t.now(), Kind: "edited", Note: strings.Join(changed, ", ")})
	if err := t.Store.Save(ctx, app); err != nil {
		return domain.Application{}, err
	}
	return app, nil
}

// Delete removes the application permanently.
func (t *Tracker) Delete(ctx context.Context, id string) error {
	if _, err := t.Store.Load(ctx, id); err != nil {
		return err
	}
	return t.Store.Delete(ctx, id)
}

// Reopen brings a terminal application back to life: to "applied" if it was
// ever submitted (AppliedAt set), else back to "drafting". Appends a
// "reopened" event. The escape hatch for a fat-fingered rejected/withdrawn or
// an auto-ghost that later got a reply.
func (t *Tracker) Reopen(ctx context.Context, id string) (domain.Application, error) {
	app, err := t.Get(ctx, id)
	if err != nil {
		return domain.Application{}, err
	}
	if !app.Status.Terminal() {
		return domain.Application{}, fmt.Errorf("%s is %s (active); reopen only applies to closed applications", id, app.Status)
	}
	to := domain.StatusDrafting
	if !app.AppliedAt.IsZero() {
		to = domain.StatusApplied
	}
	app.Status = to
	app.Events = append(app.Events, domain.Event{At: t.now(), Kind: "reopened", Note: string(to)})
	if err := t.Store.Save(ctx, app); err != nil {
		return domain.Application{}, err
	}
	return app, nil
}

// AddContact appends a contact person to the application and events it.
func (t *Tracker) AddContact(ctx context.Context, id string, c domain.AppContact) (domain.Application, error) {
	if strings.TrimSpace(c.Name) == "" {
		return domain.Application{}, fmt.Errorf("contact name is required")
	}
	app, err := t.Get(ctx, id)
	if err != nil {
		return domain.Application{}, err
	}
	app.Contacts = append(app.Contacts, c)
	app.Events = append(app.Events, domain.Event{At: t.now(), Kind: "contact", Note: c.Name})
	if err := t.Store.Save(ctx, app); err != nil {
		return domain.Application{}, err
	}
	return app, nil
}
