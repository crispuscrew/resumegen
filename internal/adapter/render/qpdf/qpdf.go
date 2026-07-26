// Package qpdf implements usecase.PDFPostProcessor by shelling out to the
// system `qpdf` binary to strip identifying metadata from a rendered PDF.
package qpdf

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Stripper removes document metadata from a PDF in place using qpdf.
//
// qpdf cannot empty an existing /Info dictionary directly, so Strip rebuilds
// the document from an empty base and imports only the page tree. That drops
// the source's /Info (/Author, /Creator, /Producer, /CreationDate, /ModDate)
// and the document-level XMP /Metadata. --linearize and
// --object-streams=disable keep the output web-optimized and diff-friendly;
// --deterministic-id avoids a time/content-seeded /ID.
type Stripper struct {
	Bin string // qpdf binary; defaults to "qpdf" when empty
}

func (s Stripper) bin() string {
	if s.Bin != "" {
		return s.Bin
	}
	return "qpdf"
}

// args returns the qpdf argument vector that rebuilds in into out with metadata
// stripped. Split out so the exact invocation can be unit-tested without qpdf.
func (s Stripper) args(in, out string) []string {
	return []string{
		"--empty",
		"--linearize",
		"--object-streams=disable",
		"--deterministic-id",
		"--pages", in, "--",
		out,
	}
}

// Strip rewrites pdfPath in place with document metadata removed. It writes to
// a sibling temp file first and renames over the original, so a qpdf failure
// leaves the input untouched.
func (s Stripper) Strip(ctx context.Context, pdfPath string) error {
	tmp, err := os.CreateTemp(filepath.Dir(pdfPath), ".qpdf-*.pdf")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	cmd := exec.CommandContext(ctx, s.bin(), s.args(pdfPath, tmpPath)...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("qpdf: %w", err)
	}
	if err := os.Rename(tmpPath, pdfPath); err != nil {
		return fmt.Errorf("replace pdf: %w", err)
	}
	return nil
}
