package loader

import (
	"github.com/crispuscrew/resumegen/internal/types"

	toml "github.com/pelletier/go-toml/v2"
)

func LoadConfig(path string) (types.Config, error) {
	path, err := resolvePath(path)
	if err != nil {return types.Config{}, err}

	return types.Config{}, nil
}
