package qpdf

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStripperArgs(t *testing.T) {
	got := Stripper{}.args("in.pdf", "out.pdf")
	want := []string{
		"--empty",
		"--linearize",
		"--object-streams=disable",
		"--deterministic-id",
		"--pages", "in.pdf", "--",
		"out.pdf",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args =\n  %v\nwant\n  %v", got, want)
	}
}

func TestStripperBinDefault(t *testing.T) {
	if got := (Stripper{}).bin(); got != "qpdf" {
		t.Errorf("default bin = %q, want qpdf", got)
	}
	if got := (Stripper{Bin: "/usr/bin/qpdf"}).bin(); got != "/usr/bin/qpdf" {
		t.Errorf("override bin = %q, want /usr/bin/qpdf", got)
	}
}

// TestStrip_Integration renders a tiny Typst document carrying a unique author
// marker, strips it, and asserts the metadata is gone. Skipped when either
// typst or qpdf is unavailable so it does not gate CI environments without them.
func TestStrip_Integration(t *testing.T) {
	typst, err := exec.LookPath("typst")
	if err != nil {
		t.Skip("typst not installed; skipping qpdf integration test")
	}
	qpdfBin, err := exec.LookPath("qpdf")
	if err != nil {
		t.Skip("qpdf not installed; skipping qpdf integration test")
	}

	dir := t.TempDir()
	const author = "ZZUNIQUEAUTHOR12345"
	src := filepath.Join(dir, "doc.typ")
	typSrc := "#set document(author: \"" + author + "\", title: \"T\", date: none)\n= Hello\nSome body text.\n"
	if err := os.WriteFile(src, []byte(typSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	pdf := filepath.Join(dir, "doc.pdf")
	if out, err := exec.Command(typst, "compile", src, pdf).CombinedOutput(); err != nil {
		t.Fatalf("typst compile: %v\n%s", err, out)
	}

	before, err := os.ReadFile(pdf)
	if err != nil {
		t.Fatal(err)
	}
	authorWasVisible := bytes.Contains(before, []byte(author)) // false if typst compressed /Info

	if err := (Stripper{}).Strip(context.Background(), pdf); err != nil {
		t.Fatalf("strip: %v", err)
	}

	after, err := os.ReadFile(pdf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(after, []byte("%PDF-")) {
		t.Fatalf("stripped output is not a PDF (head=%q)", after[:min(5, len(after))])
	}
	if authorWasVisible && bytes.Contains(after, []byte(author)) {
		t.Errorf("author marker survived stripping")
	}

	// Authoritative check: the rebuilt document has no /Info dictionary, so all
	// of /Author /Creator /Producer /CreationDate /ModDate are gone.
	trailer, err := exec.Command(qpdfBin, "--show-object=trailer", pdf).CombinedOutput()
	if err != nil {
		t.Fatalf("qpdf show trailer: %v\n%s", err, trailer)
	}
	if bytes.Contains(trailer, []byte("/Info")) {
		t.Errorf("/Info dictionary present after strip:\n%s", trailer)
	}
}
