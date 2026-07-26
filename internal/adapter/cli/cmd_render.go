package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
	"github.com/crispuscrew/resumegen/internal/adapter/container"
	"github.com/crispuscrew/resumegen/internal/adapter/render/emit"
	"github.com/crispuscrew/resumegen/internal/adapter/render/qpdf"
	"github.com/crispuscrew/resumegen/internal/adapter/tomlrepo"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

// newGenerator builds the render pipeline over the resolved appdir. It is the
// single source of the render wiring, shared by the render command and the TUI.
// bannerOut receives the container-engine banner (os.Stderr for the CLI,
// io.Discard for the TUI so it never bleeds onto the alt screen).
// layeredConfigSource resolves the active appdir and builds the config source
// with ONE layering rule for every consumer (render, tracker, prompts, TUI): in
// a walk-up workspace the global config sits under the workspace overlay;
// otherwise the resolved dir stands alone. Before v1.5's recheck the tracker
// and data-dump paths skipped the global layer, so ghost/followup/output-dir
// settings could silently differ from what render (and the TUI config screen)
// used in the same workspace.
func layeredConfigSource(appDirPath string) (usecase.ConfigSource, appdir.Resolution, error) {
	cwd, _ := os.Getwd()
	defDir := defaultAppDir()
	res, err := appdir.ResolveActive(appDirPath, cwd, defDir)
	if err != nil {
		return nil, appdir.Resolution{}, fmt.Errorf("resolve appdir: %w", err)
	}
	workspaceFS := os.DirFS(res.Dir)
	cfgSource := tomlrepo.NewLayeredConfigSource(workspaceFS, nil)
	if res.Source == appdir.SourceWalkUp {
		if defAbs, err := appdir.ExpandAbs(defDir); err == nil && defAbs != res.Dir {
			cfgSource = tomlrepo.NewLayeredConfigSource(os.DirFS(defAbs), workspaceFS)
		}
	}
	return cfgSource, res, nil
}

func newGenerator(deps Deps, appDirPath string, bannerOut io.Writer, choice func(string, bool) bool) (usecase.Generator, appdir.Resolution, error) {
	cfgSource, res, err := layeredConfigSource(appDirPath)
	if err != nil {
		return usecase.Generator{}, appdir.Resolution{}, err
	}
	workspaceFS := os.DirFS(res.Dir)

	eng, engOk := container.Detector{}.Detect()
	gen := usecase.Generator{
		Config:   cfgSource,
		Profiles: tomlrepo.NewProfileRepo(workspaceFS),
		Resumes:  tomlrepo.NewResumeRepo(workspaceFS),
		Renderer: renderRouter{
			appdir:    res.Dir,
			engine:    eng,
			engineOK:  engOk,
			image:     container.ImageTag(deps.Version),
			cfile:     deps.ContainerfileRend,
			runner:    container.ExecRunner{},
			bannerOut: bannerOut,
		},
		PostProcessor: qpdf.Stripper{},
		Emitter:       emit.Writer{},
		Bootstrap:     appdir.Bootstrap{Skeleton: deps.Skeleton, Target: res.Dir, Choice: choice},
	}
	return gen, res, nil
}

type cmdRender struct{}

func (cmdRender) Name() string { return "render" }
func (cmdRender) Synopsis() string {
	return "Render a profile (also the default when no subcommand is given)"
}

func (cmdRender) Run(ctx context.Context, deps Deps, args []string) error {
	flags := flag.NewFlagSet("resumegen render", flag.ContinueOnError)
	// Bare `resumegen --help` lands here (render is the default command), so its
	// usage must present the whole tool, not just render flags.
	flags.Usage = func() {
		printUsage(os.Stdout, topLevelCommands())
		writeln(os.Stdout, "")
		writeln(os.Stdout, "Render flags (default command):")
		flags.SetOutput(os.Stdout)
		flags.PrintDefaults()
	}
	var (
		lang        = flags.String("lang", "", "override config language")
		versionFlag = flags.Bool("version", false, "print version and exit")
		appDirPath  = flags.String("path", "", "specific path to application directory (default: walk-up from CWD, then ~/.config/resumegen/)")
		profileName = flags.String("profile", "default", "profile name to use")
		force       = flags.Bool("force", false, "render even if a bullet has malformed markup or a disallowed URL (sanitizer falls back to literal text)")
	)
	if helped, err := parseFlags(flags, args); helped || err != nil {
		return err
	}
	// flag.Parse stops at the first non-flag argument, so every flag AFTER a
	// stray positional is silently dropped: `render <profile> --path ws` used
	// to ignore --path and render the default profile from the default appdir,
	// exiting 0. render takes no positionals, so reject them.
	if flags.NArg() > 0 {
		return usageErr(fmt.Errorf("render takes no positional arguments; got %q (did you mean --profile %s?)",
			flags.Arg(0), flags.Arg(0)))
	}

	if *versionFlag {
		fmt.Printf("resumegen version: %s\n", deps.Version)
		return nil
	}

	gen, _, err := newGenerator(deps, *appDirPath, os.Stderr, UserChoice)
	if err != nil {
		return err
	}

	outPath, err := gen.Generate(ctx, usecase.GenerateInput{
		ProfileName:  *profileName,
		LangOverride: *lang,
		ForceUnsafe:  *force,
	})
	if err != nil {
		return err
	}
	fmt.Printf("rendered -> %s\n", outPath)
	return nil
}
