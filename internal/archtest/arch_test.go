// Package archtest enforces the clean-architecture import rules at test time.
// Violations fail the build via `go test ./internal/archtest/...`, so the gate
// fires on every `make test` and CI run.
package archtest

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/crispuscrew/resumegen"

// Domain must not import any other package from this module - pure types only.
func TestDomain_NoProjectImports(t *testing.T) {
	for file, imps := range collectImports(t, "../domain") {
		for _, imp := range imps {
			if strings.HasPrefix(imp, modulePath+"/") {
				t.Errorf("%s: domain imports project package %q (domain must be pure)", file, imp)
			}
		}
	}
}

// Domain must avoid IO-adjacent stdlib so it stays trivially testable and
// re-usable. The denylist captures every realistic source of IO leakage.
var domainForbiddenStdlib = map[string]struct{}{
	"os":            {},
	"os/exec":       {},
	"io":            {},
	"io/fs":         {},
	"path/filepath": {},
	"net":           {},
	"net/http":      {},
}

func TestDomain_NoIOStdlib(t *testing.T) {
	for file, imps := range collectImports(t, "../domain") {
		for _, imp := range imps {
			if _, banned := domainForbiddenStdlib[imp]; banned {
				t.Errorf("%s: domain imports IO stdlib package %q", file, imp)
			}
		}
	}
}

// Usecase orchestrates via the ports it defines; it must never know about a
// concrete adapter package.
func TestUsecase_NoAdapterImports(t *testing.T) {
	adapterPrefix := modulePath + "/internal/adapter"
	for file, imps := range collectImports(t, "../usecase") {
		for _, imp := range imps {
			if strings.HasPrefix(imp, adapterPrefix) {
				t.Errorf("%s: usecase imports adapter package %q", file, imp)
			}
		}
	}
}

// collectImports walks a directory (relative to this test file's package) and
// returns a map from .go file path to its import list. Test files are skipped.
func collectImports(t *testing.T, dir string) map[string][]string {
	t.Helper()
	result := make(map[string][]string)
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, p, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		imps := make([]string, 0, len(f.Imports))
		for _, imp := range f.Imports {
			imps = append(imps, strings.Trim(imp.Path.Value, `"`))
		}
		result[p] = imps
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	if len(result) == 0 {
		t.Fatalf("no .go files found under %s - test path is wrong", dir)
	}
	return result
}

// The notui build tag only works if charm never leaks outside adapter/tui: a
// charm import in any other package would silently fatten the "single
// dependency" notui binary. Enforced here so it cannot rot.
func TestCharm_OnlyInTUI(t *testing.T) {
	for _, dir := range []string{"../adapter", "../usecase", "../domain"} {
		for file, imps := range collectImports(t, dir) {
			if strings.Contains(filepath.ToSlash(file), "/tui/") {
				continue
			}
			for _, imp := range imps {
				if strings.HasPrefix(imp, "github.com/charmbracelet/") {
					t.Errorf("%s: charm import %q outside adapter/tui breaks the notui build posture", file, imp)
				}
			}
		}
	}
}

// The TUI must reach data only through the injected use cases - importing a
// concrete sibling adapter (trackrepo, tomlrepo, ...) would bypass the Deps
// seam that keeps it testable.
func TestTUI_NoSiblingAdapterImports(t *testing.T) {
	adapterPrefix := modulePath + "/internal/adapter/"
	for file, imps := range collectImports(t, "../adapter/tui") {
		for _, imp := range imps {
			if strings.HasPrefix(imp, adapterPrefix) && imp != modulePath+"/internal/adapter/tui" {
				t.Errorf("%s: tui imports sibling adapter %q (use the Deps seam)", file, imp)
			}
		}
	}
}
