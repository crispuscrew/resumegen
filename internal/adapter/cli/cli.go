// Package cli wires command-line flags, the user-prompt helper, and adapter
// graph construction. The binary entrypoint calls Run.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
	"github.com/crispuscrew/resumegen/internal/adapter/render/host"
	"github.com/crispuscrew/resumegen/internal/adapter/tomlrepo"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

// Deps is the runtime context injected by main: the build version string and
// the embedded skeleton filesystem used for first-run bootstrap.
type Deps struct {
	Version  string
	Skeleton fs.FS
}

// Run parses flags and either prints the version or dispatches the resume
// generation use case.
func Run(d Deps) {
	var (
		lang        = flag.String("lang", "", "override config language")
		versionFlag = flag.Bool("version", false, "print version and exit")
		appDirPath  = flag.String("path", defaultAppDir(), "specific path to application directory")
		profileName = flag.String("profile", "default", "profile name to use")
	)
	flag.Parse()

	if *versionFlag {
		fmt.Printf("resumegen version: %s\n", d.Version)
		return
	}

	resolved, err := appdir.Resolve(*appDirPath)
	if err != nil {
		log.Fatalf("Cannot resolve path: %v", err)
	}

	fsys := os.DirFS(resolved)
	gen := usecase.Generator{
		Config:    tomlrepo.NewConfigSource(fsys),
		Profiles:  tomlrepo.NewProfileRepo(fsys),
		Resumes:   tomlrepo.NewResumeRepo(fsys),
		Renderer:  host.Renderer{Appdir: resolved},
		Bootstrap: appdir.Bootstrap{Skeleton: d.Skeleton, Target: resolved, Choice: UserChoice},
	}

	outPath, err := gen.Generate(context.Background(), usecase.GenerateInput{
		ProfileName:  *profileName,
		LangOverride: *lang,
	})
	if err != nil {
		log.Fatalf("%v", err)
	}
	fmt.Printf("All is done, your output here -> %s\n", outPath)
}

func defaultAppDir() string {
	home, err := os.UserConfigDir()
	if err != nil {
		home = "." // fallback to current directory if user config dir is not available
	}
	return filepath.Join(home, "resumegen")
}

// UserChoice prompts the user for a yes/no with a default. ANSI colors are
// disabled when stdout is not a TTY or NO_COLOR is set.
func UserChoice(msg string, defaultVal bool) bool {
	var input string
	colorOn := os.Getenv("NO_COLOR") == "" && func() bool {
		fi, err := os.Stdout.Stat()
		if err != nil {
			return false
		}
		return (fi.Mode() & os.ModeCharDevice) != 0
	}()
	green := func(s string) string {
		if colorOn {
			return "\033[1;32m" + s + "\033[0m"
		}
		return s
	}
	red := func(s string) string {
		if colorOn {
			return "\033[1;31m" + s + "\033[0m"
		}
		return s
	}
	if defaultVal {
		msg += " [" + green("Y") + "/n]"
	} else {
		msg += " [y/" + red("N") + "]"
	}

	fmt.Println(msg)
	if _, err := fmt.Scanln(&input); err != nil {
		return defaultVal
	}

	switch strings.ToLower(input) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	}
	return defaultVal
}
