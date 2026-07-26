package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/crispuscrew/resumegen/internal/adapter/tui"
)

// TestCmdTUI_RefusesNonTTY asserts the TUI refuses to start when stdout is not a
// terminal (SPEC acceptance #3), returning a usage error (exit 2) rather than
// taking over a pipe. A notui build must report its stub error (exit 1) even
// when piped - the accurate diagnosis beats the generic TTY refusal. `go test`
// captures stdout via a pipe, so the non-TTY path is the norm here; skip if
// that ever isn't true.
func TestCmdTUI_RefusesNonTTY(t *testing.T) {
	if stdoutIsTerminal() {
		t.Skip("stdout is a TTY in this environment; cannot exercise the non-TTY refusal")
	}
	err := cmdTUI{}.Run(context.Background(), Deps{}, nil)
	if err == nil {
		t.Fatal("expected the TUI to refuse a non-TTY stdout")
	}
	if !tui.Supported() {
		if code := exitCode(err); code != 1 || !strings.Contains(err.Error(), "compiled without TUI support") {
			t.Fatalf("notui build should report the stub error (exit 1), got exit %d: %v", code, err)
		}
		return
	}
	if code := exitCode(err); code != 2 {
		t.Fatalf("non-TTY refusal should be a usage error (exit 2), got exit %d: %v", code, err)
	}
}
