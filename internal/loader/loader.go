package loader

import (
	"github.com/crispuscrew/resumegen/internal/model"

	"errors"
	"path/filepath"
	"os"
	"io/fs"
	"log"
)

func LoadConfiguration(appDirPath, profileName string, userChoise func(msg string, defaultVal bool) (bool)) (
	model.Config, model.ResumeData, model.Profile, string) {

	resolvedPath, err := resolvePath(appDirPath)
	if err != nil { log.Fatalf("Cannot resolve path: %v", err)}
	appDirFs := os.DirFS(resolvedPath)


	notFound := func(what string) error { return appDirSmthNotFound(what, appDirPath, userChoise) }
	tryLoad := func(err error, what string) (rerun bool) {
		if err == nil { return false }
		if errors.Is(err, fs.ErrNotExist) {
			if errors.Is(notFound(what), errRerun) { return true }
		}
		if err != nil { log.Fatalf("load error: %v", err) }
		return false
	}

	for {
		cfg, err := loadConfig(appDirFs, "config.toml")
		if tryLoad(err, "config.toml") { continue } 

		profile, err := loadProfile(appDirFs, filepath.Join("profiles", profileName+".toml"))       
		if tryLoad(err, "profile") { continue }

		data, err := loadData(appDirFs, "data")
		if tryLoad(err, "data") { continue }

		return cfg, data, profile, resolvedPath 
	}
}