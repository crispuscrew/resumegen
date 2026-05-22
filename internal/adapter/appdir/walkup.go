package appdir

import (
	"os"
	"path/filepath"
)

// MarkerSubpath is the marker file's location relative to a workspace root.
// A directory becomes a workspace by containing this file; its contents are
// metadata, not load-bearing.
const MarkerSubpath = ".resumegen/workspace.toml"

// WalkUp scans from start (which should be absolute) toward the filesystem
// root, returning the first ancestor directory that contains a workspace
// marker. The second return is false when no marker exists between start
// and the root.
func WalkUp(start string) (string, bool) {
	dir := start
	for {
		if isWorkspace(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func isWorkspace(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, MarkerSubpath))
	if err != nil {
		return false
	}
	return !info.IsDir()
}
