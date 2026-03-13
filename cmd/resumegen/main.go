package main

import (
	"github.com/crispuscrew/resumegen/internal/loader"
	"github.com/crispuscrew/resumegen/internal/score"
	"github.com/crispuscrew/resumegen/internal/render"
	"github.com/crispuscrew/resumegen/internal/guard"
	"github.com/crispuscrew/resumegen/internal/trim"

	"fmt"
	"flag"
	"strings"
	"os"
	"path/filepath"
)

var (                                                                 
	appDirPath 	= flag.String("path", defaultAppDir(), "specific path to application directory")
	profileName = flag.String("profile", "default", "profile name to use")
)

func main() {
	flag.Parse()

	cfg, data, profile, appDirPath := loader.LoadConfiguration(*appDirPath, *profileName, userChoise)
	data = score.Score(data, profile.Tags)

	var output string
	for {
		var pages float64
		output, pages = render.Render(cfg, data, profile, appDirPath)
		if !guard.TrimIsNeeded(pages, cfg.Render.PageLimit) { break }
		data = trim.TrimLowest(data)
	}
	fmt.Printf("All is done, your output here -> %s", output)
}

func userChoise(msg string, defaultVal bool) (bool) {
	var input string
	if defaultVal { msg += " [Y/n]" } else { msg += " [y/N]"}

	println(msg)
	_, err := fmt.Scanln(&input)
	if err != nil { return defaultVal }

	input = strings.ToLower(input)
	if input == "y" || input == "yes" { return true }
	if input == "n" || input == "no" { return false }
	return defaultVal
}

func defaultAppDir() (string) {
	home, err := os.UserConfigDir()
	if err != nil { home = "." } // fallback to current directory if user config dir is not available
	return filepath.Join(home, "resumegen")
}