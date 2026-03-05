package loader

import "github.com/crispuscrew/resumegen/internal/types"

func LoadConfig() (types.Config, error) {
	return types.Config{}, nil
}

func LoadData(dataDir string) (types.ResumeData, error) {
	return types.ResumeData{}, nil
}

func LoadProfile(profilesDir, name string) (types.Profile, error) {
	return types.Profile{}, nil
}
