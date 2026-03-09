package main

import (
	"github.com/crispuscrew/resumegen/internal/model"
	"github.com/crispuscrew/resumegen/internal/stage"

	"github.com/crispuscrew/resumegen/internal/loader"

	"fmt"
	"flag"
	"strings"
	"os"
	"path/filepath"
)

func main() {
	flag.Parse()
	appModel := model.Model{
		UserChoise	: userChoise,
		AppDirPath	: *appDirPath,
		ProfileName	: *profileName,
	}
	for _, stageExecute := range stages {
		var err error
		appModel, err = stage.Execute(appModel, stageExecute)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			return
		}
	}
}

var (                                                                 
	appDirPath 	= flag.String("path", defaultAppDir(), "specific path to application directory")
	profileName = flag.String("profile", "default", "profile name to use")
)

var stages = []func(model.Model) (model.Model, error){
	loader.LoadConfigStage,
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