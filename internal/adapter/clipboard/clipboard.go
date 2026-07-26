// Package clipboard copies text to the system clipboard by shelling out to a
// Wayland (wl-copy) or X11 (xclip) helper. It never opens a network socket and
// degrades to a clear error when no helper is installed.
package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// tool is one clipboard helper and the args that make it read stdin.
type tool struct {
	bin  string
	args []string
}

// order is the probe order: Wayland first, then X11.
var order = []tool{
	{bin: "wl-copy"},
	{bin: "xclip", args: []string{"-selection", "clipboard"}},
}

// lookPath is indirected for tests.
var lookPath = exec.LookPath

// Copy writes text to the clipboard using the first available helper. It returns
// a clear error if neither wl-copy nor xclip is on PATH.
func Copy(ctx context.Context, text string) error {
	for _, t := range order {
		path, err := lookPath(t.bin)
		if err != nil {
			continue
		}
		cmd := exec.CommandContext(ctx, path, t.args...)
		cmd.Stdin = bytes.NewBufferString(text)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", t.bin, err)
		}
		return nil
	}
	return fmt.Errorf("no clipboard tool found; install wl-clipboard or xclip, or use --output")
}
