package usecase_test

import (
	"context"
	"errors"
	"io/fs"
	"reflect"
	"sort"
	"testing"

	"github.com/crispuscrew/resumegen/internal/usecase"
)

// fakeExtractor records calls and answers from in-memory maps.
type fakeExtractor struct {
	files     map[string][]string // subtree -> files
	listErr   error
	already   map[string]bool // dst paths that already exist
	extracted map[string]string
}

func (f *fakeExtractor) ListSubtree(_ context.Context, subtree string) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	files, ok := f.files[subtree]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return append([]string(nil), files...), nil
}

func (f *fakeExtractor) ExtractFile(_ context.Context, srcPath, dst string) (bool, error) {
	if f.already[dst] {
		return false, nil
	}
	if f.extracted == nil {
		f.extracted = map[string]string{}
	}
	f.extracted[dst] = srcPath
	return true, nil
}

// All files copy when nothing pre-exists; only filter selects matches.
func TestExtractSubtree(t *testing.T) {
	f := &fakeExtractor{
		files: map[string][]string{
			"templates": {"templates/resume.typ", "templates/template.typ"},
		},
	}
	report, err := usecase.ExtractSubtree(context.Background(), f, "templates", "/dst", nil)
	if err != nil {
		t.Fatalf("ExtractSubtree: %v", err)
	}
	sort.Strings(report.Copied)
	want := []string{"templates/resume.typ", "templates/template.typ"}
	if !reflect.DeepEqual(report.Copied, want) {
		t.Fatalf("copied: got %v, want %v", report.Copied, want)
	}
	if len(report.Skipped) != 0 {
		t.Fatalf("skipped: got %v, want none", report.Skipped)
	}
}

// Pre-existing destination files end up in Skipped, not Copied.
func TestExtractSubtree_SkipsExisting(t *testing.T) {
	f := &fakeExtractor{
		files:   map[string][]string{"templates": {"templates/resume.typ", "templates/template.typ"}},
		already: map[string]bool{"/dst/templates/resume.typ": true},
	}
	report, err := usecase.ExtractSubtree(context.Background(), f, "templates", "/dst", nil)
	if err != nil {
		t.Fatalf("ExtractSubtree: %v", err)
	}
	if !reflect.DeepEqual(report.Copied, []string{"templates/template.typ"}) {
		t.Fatalf("copied: got %v, want [templates/template.typ]", report.Copied)
	}
	if !reflect.DeepEqual(report.Skipped, []string{"templates/resume.typ"}) {
		t.Fatalf("skipped: got %v, want [templates/resume.typ]", report.Skipped)
	}
}

// only-filter matches both bare name and basename-without-extension.
func TestExtractSubtree_NameFilter(t *testing.T) {
	f := &fakeExtractor{
		files: map[string][]string{
			"templates": {
				"templates/resume.typ",
				"templates/template.typ",
				"templates/cover.typ",
			},
		},
	}
	report, err := usecase.ExtractSubtree(context.Background(), f, "templates", "/dst",
		[]string{"resume", "cover.typ"})
	if err != nil {
		t.Fatalf("ExtractSubtree: %v", err)
	}
	sort.Strings(report.Copied)
	want := []string{"templates/cover.typ", "templates/resume.typ"}
	if !reflect.DeepEqual(report.Copied, want) {
		t.Fatalf("copied: got %v, want %v", report.Copied, want)
	}
}

// Missing subtree surfaces fs.ErrNotExist so the CLI can render a friendly
// "not in this build" message.
func TestExtractSubtree_MissingSubtreePropagatesErrNotExist(t *testing.T) {
	f := &fakeExtractor{files: map[string][]string{}}
	_, err := usecase.ExtractSubtree(context.Background(), f, "prompts", "/dst", nil)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("got %v, want fs.ErrNotExist", err)
	}
}
