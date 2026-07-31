package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
	"github.com/crispuscrew/resumegen/internal/adapter/clipboard"
	"github.com/crispuscrew/resumegen/internal/adapter/tomlrepo"
	"github.com/crispuscrew/resumegen/internal/adapter/tui"
	"github.com/crispuscrew/resumegen/internal/usecase"
	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
	"github.com/crispuscrew/resumegen/internal/usecase/track"
)

// cmdTUI launches the interactive terminal UI (v1.5). It resolves the tracker
// and render pipeline the same way `apply`/`render` do, then hands control to
// the tui adapter via injected functions so the TUI reuses CLI/use-case logic
// verbatim. Under `-tags notui` the tui package's Run is a stub, so this command
// still compiles and simply reports that TUI support was excluded.
type cmdTUI struct{}

func (cmdTUI) Name() string { return "tui" }
func (cmdTUI) Synopsis() string {
	return "Interactive terminal UI over the tracker, prompts, generate, and config (opt-in; needs a TTY)"
}

func (cmdTUI) Run(ctx context.Context, deps Deps, args []string) error {
	flags := flag.NewFlagSet("tui", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	if helped, err := parseFlags(flags, args); helped || err != nil {
		return err
	}
	// A notui build reports its stub error first, even when piped - the TTY
	// check below would otherwise mask the accurate diagnosis behind a generic
	// refusal.
	if !tui.Supported() {
		return tui.Run(tui.Deps{})
	}
	// The TUI takes over the terminal; refuse to start when stdout isn't a TTY
	// so it never garbles a pipe or a CI log (exit 2).
	if !stdoutIsTerminal() {
		return usageErr(errors.New("tui requires an interactive terminal (stdout is not a TTY)"))
	}

	tr, err := tracker(ctx, deps, *appDirPath)
	if err != nil {
		return err
	}
	// Render pipeline: discard the container banner (it would bleed onto the alt
	// screen) and never prompt for bootstrap (would fight the TUI for stdin).
	gen, res, err := newGenerator(deps, *appDirPath, io.Discard, noBootstrap)
	if err != nil {
		return err
	}

	theme := "default"
	if cfg, cerr := gen.Config.Load(ctx); cerr == nil {
		theme = cfg.TUI.ResolvedTheme()
	}

	return tui.Run(tui.Deps{
		Ctx:             ctx,
		Version:         deps.Version,
		Theme:           theme,
		Tracker:         tr,
		StaleAfterDays:  track.StaleWarnDays(tr.GhostAfterDays),
		GhostAfterDays:  tr.GhostAfterDays,
		FollowupLagDays: tr.FollowupLagDays,

		ListProfiles: func() ([]string, error) {
			return tomlrepo.ListProfiles(os.DirFS(res.Dir))
		},
		Render: func(ctx context.Context, profile string) (string, error) {
			return gen.Generate(ctx, usecase.GenerateInput{ProfileName: profile})
		},

		ListPrompts: func(ctx context.Context) ([]tui.PromptEntry, error) {
			repo, err := promptRepo(deps, *appDirPath)
			if err != nil {
				return nil, err
			}
			ents, err := repo.List(ctx)
			if err != nil {
				return nil, err
			}
			out := make([]tui.PromptEntry, 0, len(ents))
			for _, e := range ents {
				out = append(out, tui.PromptEntry{Name: e.Name, Description: e.Description, Overridden: e.Overridden})
			}
			return out, nil
		},
		LoadPrompt: func(ctx context.Context, name string) (tui.PromptForm, error) {
			repo, err := promptRepo(deps, *appDirPath)
			if err != nil {
				return tui.PromptForm{}, err
			}
			t, err := repo.Load(ctx, name)
			if err != nil {
				return tui.PromptForm{}, err
			}
			form := tui.PromptForm{Name: t.Name, Description: t.Description}
			for _, key := range sortedKeys(t.Inputs) {
				s := t.Inputs[key]
				form.Fields = append(form.Fields, tui.PromptField{
					Key: key, Source: s.Source, Flag: s.Flag, Field: s.Field,
					Default: s.Default, Required: s.Required,
				})
			}
			return form, nil
		},
		RunPrompt: func(ctx context.Context, name string, values map[string]string) (string, error) {
			repo, err := promptRepo(deps, *appDirPath)
			if err != nil {
				return "", err
			}
			t, err := repo.Load(ctx, name)
			if err != nil {
				return "", err
			}
			// Trim the synthetic fields like resolveFromValues trims regular ones —
			// a trailing space must not turn into "no data-dump at .../default .md".
			profile := strings.TrimSpace(values["__profile"])
			if profile == "" {
				profile = "default"
			}
			rc := resolveCtx{deps: deps, appdir: *appDirPath, profile: profile, appID: strings.TrimSpace(values["__app"]), noInput: true}
			in, err := rc.resolveFromValues(ctx, t, values)
			if err != nil {
				return "", err
			}
			return prompt.Render(t, in)
		},
		Copy: clipboard.Copy,

		ListDataFiles: func() ([]tui.DataFile, error) {
			return listDataFiles(res.Dir)
		},
		EditCmd: editCmd,

		LoadConfig: func(ctx context.Context) (tui.ConfigView, error) {
			return loadConfigView(ctx, gen, res)
		},
	})
}

// noBootstrap is the bootstrap Choice used by the TUI: never confirm (so an
// uninitialized appdir yields a clean render error rather than a stdin prompt
// that would deadlock against bubbletea).
func noBootstrap(string, bool) bool { return false }

func listDataFiles(appdirDir string) ([]tui.DataFile, error) {
	dataDir := filepath.Join(appdirDir, "data")
	ents, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []tui.DataFile
	for _, e := range ents {
		// Skip dirs, non-TOML, and dotfiles (editor lock files like `.#x.toml`).
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		out = append(out, tui.DataFile{Name: e.Name(), Path: filepath.Join(dataDir, e.Name())})
	}
	return out, nil
}

func editCmd(path string) *exec.Cmd {
	// Split so EDITOR="code -w" style values work; args after the program pass
	// through. Fields of a whitespace-only EDITOR is empty — fall back to vi
	// rather than panic on fields[0].
	fields := strings.Fields(os.Getenv("EDITOR"))
	if len(fields) == 0 {
		fields = []string{"vi"}
	}
	prog := fields[0]
	rest := append(fields[1:], path)
	return exec.Command(prog, rest...)
}

func loadConfigView(ctx context.Context, gen usecase.Generator, res appdir.Resolution) (tui.ConfigView, error) {
	cfg, err := gen.Config.Load(ctx)
	if err != nil {
		return tui.ConfigView{}, err
	}
	tc := cfg.Tracker.WithDefaults()
	return tui.ConfigView{
		Appdir: res.Dir,
		Origin: originName(res.Source),
		Lines: []tui.ConfigLine{
			{Key: "paths.output_dir", Value: cfg.Paths.OutputDir},
			{Key: "paths.typst_bin", Value: cfg.Paths.TypstBin},
			{Key: "render.page_limit", Value: fmt.Sprintf("%g", cfg.Render.PageLimit)},
			{Key: "render.page_height_pt", Value: fmt.Sprintf("%g", cfg.Render.PageHeightPt)},
			{Key: "render.use_container", Value: valueOr(cfg.Render.UseContainer, "(host)")},
			{Key: "render.strip_metadata", Value: fmt.Sprintf("%t", cfg.Render.StripMetadata)},
			{Key: "render.force_unsafe", Value: fmt.Sprintf("%t", cfg.Render.ForceUnsafe)},
			{Key: "score.skill_priority", Value: fmt.Sprintf("%d", cfg.Score.SkillPriority)},
			{Key: "tracker.ghost_after_days", Value: fmt.Sprintf("%d", tc.GhostAfterDays)},
			{Key: "tracker.followup_default_lag_days", Value: fmt.Sprintf("%d", tc.FollowupDefaultLagDays)},
		},
	}, nil
}

func valueOr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func originName(s appdir.ResolutionSource) string {
	switch s {
	case appdir.SourceFlag:
		return "flag (--path)"
	case appdir.SourceWalkUp:
		return "workspace marker (walk-up)"
	default:
		return "default (~/.config/resumegen)"
	}
}

func stdoutIsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
