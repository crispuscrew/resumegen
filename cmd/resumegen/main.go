package main

import (
	"github.com/crispuscrew/resumegen/internal/guard"
	"github.com/crispuscrew/resumegen/internal/loader"
	"github.com/crispuscrew/resumegen/internal/render"
	"github.com/crispuscrew/resumegen/internal/score"
	"github.com/crispuscrew/resumegen/internal/trim"

	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (                       
	version 	= "dev" // set by build process
	lang 		= flag.String("lang", "", "override config language")
	versionFlag = flag.Bool("version", false, "print version and exit")
	appDirPath 	= flag.String("path", defaultAppDir(), "specific path to application directory")
	profileName = flag.String("profile", "default", "profile name to use")
)

func main() {
	flag.Parse()
	if *versionFlag { fmt.Printf("resumegen version: %s\n", version); return }
	
	cfg, data, profile, appDirPath := loader.LoadConfiguration(*appDirPath, *profileName, userChoise)
	data = score.Score(data, profile.Tags, cfg.Score)
	if *lang != "" { profile.Lang = *lang }

	var output string
	for {
		var pages float64; var trimmed bool
		output, pages = render.Render(cfg, data, profile, appDirPath)
		if !guard.TrimIsNeeded(pages, cfg.Render.PageLimit) { break }
		data, trimmed = trim.TrimLowest(data, cfg.Render.MinElements)
		if !trimmed { log.Fatalf("Cannot trim anymore, but page limit is still not met. Please check your data and profile settings.") }
	}
	fmt.Printf("All is done, your output here -> %s\n", output)
}

func userChoise(msg string, defaultVal bool) (bool) {
	var input string
	var colorOn = os.Getenv("NO_COLOR") == "" && func() bool {
		fi, err := os.Stdout.Stat()
		if err != nil { log.Fatal(err) }
		return (fi.Mode() & os.ModeCharDevice) != 0
	}()
	green 	:= func(s string) string {if colorOn { return "\033[1;32m" + s + "\033[0m" }; return s}
	red 	:= func(s string) string {if colorOn { return "\033[1;31m" + s + "\033[0m" }; return s}
	if defaultVal { 
		msg += " [" + green("Y") + "/n]" 
	} else { 
		msg += " [y/" + red("N") + "]"
	}

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