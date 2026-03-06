package loader

import (
	"os"
	"path/filepath"
	"strings"
)

func resolvePath(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {return "", err}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	} else if strings.HasPrefix(path, "/") {
	} else {
		path = filepath.Abs(path)
	}
	return path, nil
}