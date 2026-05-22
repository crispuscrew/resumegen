// Package cli wires command-line flags, subcommand dispatch, and the
// user-prompt helper. The binary entrypoint constructs Deps and calls Run
// (defined in router.go).
package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Deps is the runtime context injected by main: the build version string,
// the embedded skeleton filesystem used for first-run bootstrap and the
// `init` / `extract` subcommands, and the embedded Containerfile used to
// build the local render image on demand (slice 4).
type Deps struct {
	Version           string
	Skeleton          fs.FS
	ContainerfileRend []byte
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
