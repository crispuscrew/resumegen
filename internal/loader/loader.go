package loader

import (
	"github.com/crispuscrew/resumegen/internal/model"

	"errors"
	"path/filepath"
	"os"
	"io/fs"
)

func LoadConfigStage(mdl model.Model) (model.Model, error) {
	resolvedPath, err := resolvePath(mdl.AppDirPath)
	if err != nil { return mdl, err }
	mdl.AppDirFs = os.DirFS(resolvedPath)

	cfg, err := loadConfig(mdl.AppDirFs, "config.toml")
	if errors.Is(err, fs.ErrNotExist) {
		return appDirSmthNotFound(mdl, "Config file")
	} else if err != nil { return mdl, err }

	mdl.Cfg = cfg
	return mdl, nil
}

func LoadProfileStage(mdl model.Model) (model.Model, error) {
	profile, err := loadProfile(mdl.AppDirFs, filepath.Join("profiles", mdl.ProfileName+".toml"))
	if err != nil { return mdl, err }

	mdl.Profile = profile
	return mdl, nil
}

func LoadDataStage(mdl model.Model) (model.Model, error) {
	data, err := loadData(mdl.AppDirFs, "data")
	if errors.Is(err, fs.ErrNotExist) {
		return appDirSmthNotFound(mdl, "Data files")
	} else if err != nil { return mdl, err }

	mdl.Data = data
	return mdl, nil
}