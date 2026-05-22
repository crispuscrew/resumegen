package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
)

// Init on an empty dir with a small skeleton should create the marker plus
// every default-tier file, leave a re-run as a no-op for data/profiles, and
// surface --full-example as the trigger to also copy templates.
func TestCmdInit_EndToEnd(t *testing.T) {
	skeleton := fstest.MapFS{
		"config.toml":            &fstest.MapFile{Data: []byte("# skeleton config")},
		"data/header.toml":       &fstest.MapFile{Data: []byte("name = \"x\"")},
		"profiles/default.toml":  &fstest.MapFile{Data: []byte("tags = []")},
		"templates/resume.typ":   &fstest.MapFile{Data: []byte("// resume template")},
	}
	deps := Deps{Version: "test", Skeleton: skeleton}
	target := t.TempDir()
	ctx := context.Background()

	// 1) Default init: marker + data + profiles, no templates.
	if err := (cmdInit{}).Run(ctx, deps, []string{"--name", "test-ws", target}); err != nil {
		t.Fatalf("init (default): %v", err)
	}
	mustExist(t, filepath.Join(target, appdir.MarkerSubpath))
	mustExist(t, filepath.Join(target, "data", "header.toml"))
	mustExist(t, filepath.Join(target, "profiles", "default.toml"))
	mustNotExist(t, filepath.Join(target, "templates", "resume.typ"))

	// 2) Re-run: marker remains; no error; existing files untouched.
	headerBefore := readFile(t, filepath.Join(target, "data", "header.toml"))
	if err := (cmdInit{}).Run(ctx, deps, []string{target}); err != nil {
		t.Fatalf("init (rerun): %v", err)
	}
	headerAfter := readFile(t, filepath.Join(target, "data", "header.toml"))
	if headerBefore != headerAfter {
		t.Fatalf("data/header.toml mutated on re-run: before=%q after=%q", headerBefore, headerAfter)
	}

	// 3) --full-example pulls templates in too.
	if err := (cmdInit{}).Run(ctx, deps, []string{"--full-example", target}); err != nil {
		t.Fatalf("init (--full-example): %v", err)
	}
	mustExist(t, filepath.Join(target, "templates", "resume.typ"))

	// 4) Marker contents reflect the --name we passed on the first call.
	repo := appdir.NewWorkspaceRepo()
	ws, err := repo.Load(ctx, target)
	if err != nil {
		t.Fatalf("Load marker: %v", err)
	}
	if ws.Workspace.Name != "test-ws" {
		t.Fatalf("marker name: got %q, want %q", ws.Workspace.Name, "test-ws")
	}
}

// --bare creates the marker and nothing else.
func TestCmdInit_Bare(t *testing.T) {
	skeleton := fstest.MapFS{
		"data/header.toml":      &fstest.MapFile{Data: []byte("name = \"x\"")},
		"profiles/default.toml": &fstest.MapFile{Data: []byte("tags = []")},
	}
	deps := Deps{Version: "test", Skeleton: skeleton}
	target := t.TempDir()

	if err := (cmdInit{}).Run(context.Background(), deps, []string{"--bare", target}); err != nil {
		t.Fatalf("init --bare: %v", err)
	}
	mustExist(t, filepath.Join(target, appdir.MarkerSubpath))
	mustNotExist(t, filepath.Join(target, "data", "header.toml"))
	mustNotExist(t, filepath.Join(target, "profiles", "default.toml"))
}

func mustExist(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected %s to exist: %v", p, err)
	}
}

func mustNotExist(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err == nil {
		t.Fatalf("expected %s NOT to exist", p)
	} else if !isNotExist(err) {
		t.Fatalf("stat %s: %v", p, err)
	}
}

func isNotExist(err error) bool { return err != nil && os.IsNotExist(err) }

func readFile(t *testing.T, p string) string {
	t.Helper()
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(raw)
}

// compile-time: ensure cmdInit satisfies the Command interface.
var _ Command = cmdInit{}
