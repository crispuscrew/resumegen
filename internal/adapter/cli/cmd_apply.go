package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/crispuscrew/resumegen/internal/adapter/trackrepo"
	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase"
	"github.com/crispuscrew/resumegen/internal/usecase/track"
)

// dateLayout is the only accepted date format for --due and friends: a TOML
// local-date. Other formats are a usage error (exit 2).
const dateLayout = "2006-01-02"

type cmdApply struct{}

func (cmdApply) Name() string { return "apply" }
func (cmdApply) Synopsis() string {
	return "Track where you applied and what happened (records only; never renders or calls an LLM)"
}

func (cmdApply) Run(ctx context.Context, deps Deps, args []string) error {
	if len(args) == 0 {
		return usageErr(errors.New("usage: resumegen apply <new|list|show|set|edit|followup|note|contact|reopen|delete> [args]"))
	}
	switch args[0] {
	case "new":
		return applyNew(ctx, deps, args[1:])
	case "list":
		return applyList(ctx, deps, args[1:])
	case "show":
		return applyShow(ctx, deps, args[1:])
	case "set":
		return applySet(ctx, deps, args[1:])
	case "followup":
		return applyFollowup(ctx, deps, args[1:])
	case "note":
		return applyNote(ctx, deps, args[1:])
	case "edit":
		return applyEdit(ctx, deps, args[1:])
	case "delete":
		return applyDelete(ctx, deps, args[1:])
	case "reopen":
		return applyReopen(ctx, deps, args[1:])
	case "contact":
		return applyContact(ctx, deps, args[1:])
	case "help", "-h", "--help":
		printApplyHelp()
		return nil
	default:
		return usageErr(fmt.Errorf("unknown apply subcommand: %s", args[0]))
	}
}

func printApplyHelp() {
	writeln(os.Stdout, "Usage: resumegen apply <subcommand> [args]")
	writeln(os.Stdout, "  new       Create a drafting application (--company --role [--profile --jd --source --status])")
	writeln(os.Stdout, "  list      List applications (--status --stale --followups-due --json)")
	writeln(os.Stdout, "  show      Show one application (--json)")
	writeln(os.Stdout, "  set       Change status: apply set <id> status <status> [--note]")
	writeln(os.Stdout, "  edit      Fix fields: apply edit <id> --company/--role/--salary/--jd/--source/--profile/--remote/--resume-pdf/--cover-letter/--notes")
	writeln(os.Stdout, "  followup  Add one: apply followup <id> --action <text> [--due <YYYY-MM-DD>]; complete one: --done <n>")
	writeln(os.Stdout, "  note      Append a note: apply note <id> [--path <dir>] <text>")
	writeln(os.Stdout, "  contact   Record a person: apply contact <id> --name <n> [--role --channel]")
	writeln(os.Stdout, "  reopen    Bring a closed application back (rejected/withdrawn/ghosted -> active)")
	writeln(os.Stdout, "  delete    Remove an application permanently: apply delete <id> [--yes]")
}

// tracker builds a track.Tracker over the resolved appdir, reading tracker
// config (with defaults, global-under-workspace layering) and a real clock.
// A missing config falls back to defaults, but a config that EXISTS and fails
// to parse is a loud error — silently ghosting at 30 days because the user
// typo'd their 90 would be a debugging nightmare.
func tracker(ctx context.Context, deps Deps, appDirPath string) (*track.Tracker, error) {
	cfgSource, res, err := layeredConfigSource(appDirPath)
	if err != nil {
		return nil, err
	}
	tc := domain.Tracker{}
	cfg, cerr := cfgSource.Load(ctx)
	switch {
	case cerr == nil:
		tc = cfg.Tracker
	case errors.Is(cerr, usecase.ErrWorkspaceMissing):
		// no config.toml anywhere: pure defaults, fine for a fresh dir
	default:
		return nil, fmt.Errorf("config: %w", cerr)
	}
	tc = tc.WithDefaults()

	newStore := deps.NewStore
	if newStore == nil {
		newStore = trackrepo.New
	}
	return &track.Tracker{
		Store:           newStore(res.Dir),
		Now:             time.Now,
		GhostAfterDays:  tc.GhostAfterDays,
		FollowupLagDays: tc.FollowupDefaultLagDays,
	}, nil
}

func applyNew(ctx context.Context, deps Deps, args []string) error {
	flags := flag.NewFlagSet("apply new", flag.ContinueOnError)
	var (
		appDirPath = flags.String("path", "", "specific path to application directory")
		company    = flags.String("company", "", "hiring company (required)")
		role       = flags.String("role", "", "role/title (required)")
		profile    = flags.String("profile", "", "resume profile used")
		jd         = flags.String("jd", "", "path to a JD text file you keep (stored, never fetched)")
		source     = flags.String("source", "", "where the JD came from (free text)")
		status     = flags.String("status", "", "initial status (default drafting; advances through the chain)")
		salary     = flags.String("salary", "", "salary range (free text)")
		remote     = flags.Bool("remote", false, "mark the role as remote")
		jsonOut    = flags.Bool("json", false, "emit the created application as JSON")
	)
	if helped, err := parseFlags(flags, args); helped || err != nil {
		return err
	}
	if *company == "" || *role == "" {
		return usageErr(errors.New("apply new requires --company and --role"))
	}
	if *status != "" && !domain.Status(*status).Valid() {
		// Same class of mistake as `apply set` with a bad status: usage error.
		return usageErr(&domain.InvalidStatusError{Value: *status})
	}
	tr, err := tracker(ctx, deps, *appDirPath)
	if err != nil {
		return err
	}
	app, err := tr.New(ctx, track.NewInput{
		Company: *company, Role: *role, Profile: *profile, JDPath: *jd,
		Source: *source, SalaryRange: *salary, Remote: *remote,
		Status: domain.Status(*status),
	})
	if err != nil {
		return resolutionErr(err)
	}
	if *jsonOut {
		return emitJSON(showApplicationJSON(app))
	}
	writef(os.Stdout, "created %s\n", app.ID)
	return nil
}

func applyList(ctx context.Context, deps Deps, args []string) error {
	flags := flag.NewFlagSet("apply list", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	statusCSV := flags.String("status", "", "keep only these statuses (comma-separated)")
	stale := flags.Int("stale", 0, "keep entries with no activity in more than N days")
	followupsDue := flags.Bool("followups-due", false, "keep entries with a followup due today or earlier")
	jsonOut := flags.Bool("json", false, "emit a stable JSON array")
	if helped, err := parseFlags(flags, args); helped || err != nil {
		return err
	}

	q := track.Query{FollowupsDue: *followupsDue}
	if *statusCSV != "" {
		for _, s := range strings.Split(*statusCSV, ",") {
			st := domain.Status(strings.TrimSpace(s))
			if !st.Valid() {
				return usageErr(&domain.InvalidStatusError{Value: string(st)})
			}
			q.Statuses = append(q.Statuses, st)
		}
	}
	if *stale > 0 {
		q = q.WithStaleDays(*stale)
	}

	tr, err := tracker(ctx, deps, *appDirPath)
	if err != nil {
		return err
	}
	apps, err := tr.List(ctx, q)
	if err != nil {
		return resolutionErr(err)
	}
	if *jsonOut {
		return emitJSON(listApplicationsJSON(apps, tr.Now()))
	}
	printApplicationList(apps)
	return nil
}

// printApplicationList renders the human-readable table for `apply list`.
func printApplicationList(apps []domain.Application) {
	if len(apps) == 0 {
		writeln(os.Stdout, "no applications found")
		return
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	writef(tw, "ID\tCOMPANY\tROLE\tSTATUS\tAPPLIED\tDUE\n")
	for _, a := range apps {
		applied := ""
		if !a.AppliedAt.IsZero() {
			applied = a.AppliedAt.Format(dateLayout)
		}
		due := ""
		if d := a.NextFollowupDue(); !d.IsZero() {
			due = d.Format(dateLayout)
		}
		writef(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", a.ID, a.Company, a.Role, a.Status, applied, due)
	}
	_ = tw.Flush()
}

func applyShow(ctx context.Context, deps Deps, args []string) error {
	if helped, err := positionalArgs(args, 1, "resumegen apply show <id> [--json]"); helped || err != nil {
		return err
	}
	id := args[0]
	flags := flag.NewFlagSet("apply show", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	jsonOut := flags.Bool("json", false, "emit the full application as JSON")
	if helped, err := parseFlags(flags, args[1:]); helped || err != nil {
		return err
	}
	tr, err := tracker(ctx, deps, *appDirPath)
	if err != nil {
		return err
	}
	app, err := tr.Get(ctx, id)
	if err != nil {
		return resolutionErr(err)
	}
	if *jsonOut {
		return emitJSON(showApplicationJSON(app))
	}
	printApplication(app)
	return nil
}

func applySet(ctx context.Context, deps Deps, args []string) error {
	// Grammar: apply set <id> status <status> [--note <text>]
	if helped, err := positionalArgs(args, 3, "resumegen apply set <id> status <status> [--note <text>]"); helped || err != nil {
		return err
	}
	id, field, value := args[0], args[1], args[2]
	if field != "status" {
		return usageErr(fmt.Errorf("apply set: only 'status' is settable, got %q", field))
	}
	flags := flag.NewFlagSet("apply set", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	note := flags.String("note", "", "note attached to the status-change event")
	jsonOut := flags.Bool("json", false, "emit the updated application as JSON")
	if helped, err := parseFlags(flags, args[3:]); helped || err != nil {
		return err
	}
	to := domain.Status(value)
	if !to.Valid() {
		return usageErr(&domain.InvalidStatusError{Value: value})
	}
	tr, err := tracker(ctx, deps, *appDirPath)
	if err != nil {
		return err
	}
	app, err := tr.Set(ctx, id, to, *note)
	if err != nil {
		return resolutionErr(err)
	}
	if *jsonOut {
		return emitJSON(showApplicationJSON(app))
	}
	writef(os.Stdout, "%s -> %s\n", app.ID, app.Status)
	return nil
}

func applyFollowup(ctx context.Context, deps Deps, args []string) error {
	if helped, err := positionalArgs(args, 1, "resumegen apply followup <id> --action <text> [--due <YYYY-MM-DD>] | --done <n>"); helped || err != nil {
		return err
	}
	id := args[0]
	flags := flag.NewFlagSet("apply followup", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	due := flags.String("due", "", "due date YYYY-MM-DD (default: today + followup_default_lag_days)")
	action := flags.String("action", "", "what to do (required when adding)")
	done := flags.Int("done", 0, "mark the n-th followup (as listed by `apply show`) done instead of adding")
	jsonOut := flags.Bool("json", false, "emit the updated application as JSON")
	if helped, err := parseFlags(flags, args[1:]); helped || err != nil {
		return err
	}
	tr, err := tracker(ctx, deps, *appDirPath)
	if err != nil {
		return err
	}
	if *done > 0 {
		if *action != "" || *due != "" {
			return usageErr(errors.New("apply followup: --done cannot be combined with --action/--due"))
		}
		app, err := tr.CompleteFollowup(ctx, id, *done)
		if err != nil {
			return resolutionErr(err)
		}
		if *jsonOut {
			return emitJSON(showApplicationJSON(app))
		}
		writef(os.Stdout, "%s followup %d done\n", app.ID, *done)
		return nil
	}
	if *action == "" {
		return resolutionErr(errors.New("apply followup requires --action (or --done <n> to complete one)"))
	}
	dueAt, err := resolveDue(*due)
	if err != nil {
		return err
	}
	app, err := tr.AddFollowup(ctx, id, dueAt, *action)
	if err != nil {
		return resolutionErr(err)
	}
	if *jsonOut {
		return emitJSON(showApplicationJSON(app))
	}
	added := app.Followups[len(app.Followups)-1] // AddFollowup appends; its Due carries the applied default lag
	writef(os.Stdout, "%s followup due %s: %s\n", app.ID, added.Due.Format(dateLayout), *action)
	return nil
}

func applyNote(ctx context.Context, deps Deps, args []string) error {
	if helped, err := positionalArgs(args, 2, "resumegen apply note <id> [--path <dir>] <text>"); helped || err != nil {
		return err
	}
	id := args[0]
	// Parse flags before the free-text tail, so --path works like on every
	// other apply subcommand (it used to be silently swallowed into the note).
	flags := flag.NewFlagSet("apply note", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	words, helped, err := parseFlagsInterleaved(flags, args[1:])
	if helped || err != nil {
		return err
	}
	text := strings.Join(words, " ")
	if strings.TrimSpace(text) == "" {
		return usageErr(errors.New("apply note: note text is required"))
	}
	tr, err := tracker(ctx, deps, *appDirPath)
	if err != nil {
		return err
	}
	app, err := tr.AddNote(ctx, id, text)
	if err != nil {
		return resolutionErr(err)
	}
	writef(os.Stdout, "%s note recorded\n", app.ID)
	return nil
}

// resolveDue parses --due as a local-date. Empty means "use the default lag" —
// signalled to the tracker as the zero time; the lag rule lives in ONE place
// (track.AddFollowup), not per front-end.
func resolveDue(due string) (time.Time, error) {
	if due == "" {
		return time.Time{}, nil
	}
	d, err := time.Parse(dateLayout, due)
	if err != nil {
		return time.Time{}, usageErr(fmt.Errorf("invalid --due %q: want YYYY-MM-DD", due))
	}
	return d, nil
}

// printApplication renders the human-readable labelled block for `apply show`.
func printApplication(a domain.Application) {
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	writef(tw, "ID:\t%s\n", a.ID)
	writef(tw, "Company:\t%s\n", a.Company)
	writef(tw, "Role:\t%s\n", a.Role)
	writef(tw, "Status:\t%s\n", a.Status)
	if a.Profile != "" {
		writef(tw, "Profile:\t%s\n", a.Profile)
	}
	if a.Source != "" {
		writef(tw, "Source:\t%s\n", a.Source)
	}
	if !a.AppliedAt.IsZero() {
		writef(tw, "Applied:\t%s\n", a.AppliedAt.Format(dateLayout))
	}
	if a.JDPath != "" {
		writef(tw, "JD path:\t%s\n", a.JDPath)
	}
	if a.SalaryRange != "" {
		writef(tw, "Salary:\t%s\n", a.SalaryRange)
	}
	if a.Remote {
		writef(tw, "Remote:\t%t\n", a.Remote)
	}
	if a.Notes != "" {
		writef(tw, "Notes:\t%s\n", a.Notes)
	}
	_ = tw.Flush()

	if len(a.Followups) > 0 {
		writeln(os.Stdout, "\nFollowups:")
		ftw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for i, f := range a.Followups {
			done := " "
			if f.Done {
				done = "x"
			}
			writef(ftw, "  %d.\t[%s]\t%s\t%s\n", i+1, done, f.Due.Format(dateLayout), f.Action)
		}
		_ = ftw.Flush()
	}
	if len(a.Contacts) > 0 {
		writeln(os.Stdout, "\nContacts:")
		ctw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, c := range a.Contacts {
			writef(ctw, "  %s\t%s\t%s\t%s\n", c.Name, c.Role, c.Channel, dayString(c.LastContact))
		}
		_ = ctw.Flush()
	}
	if len(a.Events) > 0 {
		writeln(os.Stdout, "\nEvents:")
		etw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, e := range a.Events {
			writef(etw, "  %s\t%s\t%s\n", e.At.Format("2006-01-02 15:04"), e.Kind, e.Note)
		}
		_ = etw.Flush()
	}
}

// applyEdit fixes fields after creation. Flags left unset stay untouched
// (flag.Visit distinguishes "not given" from "given empty"); the ID never
// changes — it is stable identity, not derived state.
func applyEdit(ctx context.Context, deps Deps, args []string) error {
	if helped, err := positionalArgs(args, 1, "resumegen apply edit <id> [--company ...] [--role ...] [--salary ...] [--jd ...] [--source ...] [--profile ...] [--remote=true|false] [--resume-pdf ...] [--cover-letter ...] [--notes ...]"); helped || err != nil {
		return err
	}
	id := args[0]
	flags := flag.NewFlagSet("apply edit", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	company := flags.String("company", "", "hiring company")
	role := flags.String("role", "", "role/title")
	profile := flags.String("profile", "", "resume profile used")
	jd := flags.String("jd", "", "path to the JD text file")
	source := flags.String("source", "", "where the JD came from")
	salary := flags.String("salary", "", "salary range")
	resumePDF := flags.String("resume-pdf", "", "path of the resume PDF you sent")
	coverLetter := flags.String("cover-letter", "", "path of the cover letter you sent")
	notes := flags.String("notes", "", "replace the notes field")
	remote := flags.Bool("remote", false, "mark the role as remote (use --remote=false to unset)")
	jsonOut := flags.Bool("json", false, "emit the updated application as JSON")
	if helped, err := parseFlags(flags, args[1:]); helped || err != nil {
		return err
	}
	in := track.EditInput{}
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "company":
			in.Company = company
		case "role":
			in.Role = role
		case "profile":
			in.Profile = profile
		case "jd":
			in.JDPath = jd
		case "source":
			in.Source = source
		case "salary":
			in.SalaryRange = salary
		case "resume-pdf":
			in.ResumePDF = resumePDF
		case "cover-letter":
			in.CoverLetter = coverLetter
		case "notes":
			in.Notes = notes
		case "remote":
			in.Remote = remote
		}
	})
	tr, err := tracker(ctx, deps, *appDirPath)
	if err != nil {
		return err
	}
	app, err := tr.Edit(ctx, id, in)
	if err != nil {
		return resolutionErr(err)
	}
	if *jsonOut {
		return emitJSON(showApplicationJSON(app))
	}
	writef(os.Stdout, "%s updated\n", app.ID)
	return nil
}

// applyDelete removes an application permanently. Requires --yes, or an
// interactive confirmation on a real terminal (headless without --yes fails
// safe and deletes nothing).
func applyDelete(ctx context.Context, deps Deps, args []string) error {
	if helped, err := positionalArgs(args, 1, "resumegen apply delete <id> [--yes]"); helped || err != nil {
		return err
	}
	id := args[0]
	flags := flag.NewFlagSet("apply delete", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	yes := flags.Bool("yes", false, "delete without asking")
	if helped, err := parseFlags(flags, args[1:]); helped || err != nil {
		return err
	}
	tr, err := tracker(ctx, deps, *appDirPath)
	if err != nil {
		return err
	}
	if !*yes && !UserChoice(fmt.Sprintf("permanently delete %s?", id), false) {
		return resolutionErr(fmt.Errorf("not deleted (pass --yes to skip the confirmation)"))
	}
	if err := tr.Delete(ctx, id); err != nil {
		return resolutionErr(err)
	}
	writef(os.Stdout, "%s deleted\n", id)
	return nil
}

// applyReopen brings a closed application back to life.
func applyReopen(ctx context.Context, deps Deps, args []string) error {
	if helped, err := positionalArgs(args, 1, "resumegen apply reopen <id>"); helped || err != nil {
		return err
	}
	id := args[0]
	flags := flag.NewFlagSet("apply reopen", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	jsonOut := flags.Bool("json", false, "emit the updated application as JSON")
	if helped, err := parseFlags(flags, args[1:]); helped || err != nil {
		return err
	}
	tr, err := tracker(ctx, deps, *appDirPath)
	if err != nil {
		return err
	}
	app, err := tr.Reopen(ctx, id)
	if err != nil {
		return resolutionErr(err)
	}
	if *jsonOut {
		return emitJSON(showApplicationJSON(app))
	}
	writef(os.Stdout, "%s reopened -> %s\n", app.ID, app.Status)
	return nil
}

// applyContact records a person tied to an application.
func applyContact(ctx context.Context, deps Deps, args []string) error {
	if helped, err := positionalArgs(args, 1, "resumegen apply contact <id> --name <name> [--role <r>] [--channel <c>]"); helped || err != nil {
		return err
	}
	id := args[0]
	flags := flag.NewFlagSet("apply contact", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	name := flags.String("name", "", "person's name (required)")
	role := flags.String("role", "", "their role (recruiter, hiring manager, ...)")
	channel := flags.String("channel", "", "how you talk (email, telegram, ...)")
	jsonOut := flags.Bool("json", false, "emit the updated application as JSON")
	if helped, err := parseFlags(flags, args[1:]); helped || err != nil {
		return err
	}
	if *name == "" {
		return usageErr(errors.New("apply contact requires --name"))
	}
	tr, err := tracker(ctx, deps, *appDirPath)
	if err != nil {
		return err
	}
	app, err := tr.AddContact(ctx, id, domain.AppContact{
		Name: *name, Role: *role, Channel: *channel, LastContact: tr.Now(),
	})
	if err != nil {
		return resolutionErr(err)
	}
	if *jsonOut {
		return emitJSON(showApplicationJSON(app))
	}
	writef(os.Stdout, "%s contact recorded: %s\n", app.ID, *name)
	return nil
}
