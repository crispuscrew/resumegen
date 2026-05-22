package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
	"github.com/crispuscrew/resumegen/internal/adapter/render/host"
	"github.com/crispuscrew/resumegen/internal/adapter/tomlrepo"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

type cmdRender struct{}

func (cmdRender) Name() string     { return "render" }
func (cmdRender) Synopsis() string { return "Render a profile (also the default when no subcommand is given)" }

func (cmdRender) Run(ctx context.Context, deps Deps, args []string) error {
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	var (
		lang        = flags.String("lang", "", "override config language")
		versionFlag = flags.Bool("version", false, "print version and exit")
		appDirPath  = flags.String("path", "", "specific path to application directory (default: walk-up from CWD, then ~/.config/resumegen/)")
		profileName = flags.String("profile", "default", "profile name to use")
		force       = flags.Bool("force", false, "render even if a bullet has malformed markup or a disallowed URL (sanitizer falls back to literal text)")
	)
	if err := flags.Parse(args); err != nil {
		return err
	}

	if *versionFlag {
		fmt.Printf("resumegen version: %s\n", deps.Version)
		return nil
	}

	cwd, _ := os.Getwd()
	defDir := defaultAppDir()
	res, err := appdir.ResolveActive(*appDirPath, cwd, defDir)
	if err != nil {
		return fmt.Errorf("resolve appdir: %w", err)
	}

	workspaceFS := os.DirFS(res.Dir)
	cfgSource := tomlrepo.NewLayeredConfigSource(workspaceFS, nil)
	if res.Source == appdir.SourceWalkUp {
		if defAbs, err := appdir.ExpandAbs(defDir); err == nil && defAbs != res.Dir {
			cfgSource = tomlrepo.NewLayeredConfigSource(os.DirFS(defAbs), workspaceFS)
		}
	}

	gen := usecase.Generator{
		Config:    cfgSource,
		Profiles:  tomlrepo.NewProfileRepo(workspaceFS),
		Resumes:   tomlrepo.NewResumeRepo(workspaceFS),
		Renderer:  host.Renderer{Appdir: res.Dir},
		Bootstrap: appdir.Bootstrap{Skeleton: deps.Skeleton, Target: res.Dir, Choice: UserChoice},
	}

	outPath, err := gen.Generate(ctx, usecase.GenerateInput{
		ProfileName:  *profileName,
		LangOverride: *lang,
		ForceUnsafe:  *force,
	})
	if err != nil {
		return err
	}
	fmt.Printf("All is done, your output here -> %s\n", outPath)
	return nil
}
