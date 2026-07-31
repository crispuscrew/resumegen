// Package emit implements usecase.ResumeEmitter: it writes machine-readable
// siblings of the rendered PDF - a Markdown dump and the filtered TOML - so the
// exact filtered resume can be fed to an LLM. The PDF itself is never touched.
package emit

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// The dumps carry the full resume (PII): private by default.
const filePerm fs.FileMode = 0o600

// writePrivate writes data to path with filePerm, enforcing the mode even when
// path already exists. os.WriteFile applies perm only at creation, so a dump
// left over from a build before the 0600 hardening would keep its 0644 and be
// silently refilled with PII.
func writePrivate(path string, data []byte) error {
	if err := os.WriteFile(path, data, filePerm); err != nil {
		return err
	}
	return os.Chmod(path, filePerm)
}

// Writer derives sibling paths from the PDF path and writes the enabled
// outputs. It satisfies usecase.ResumeEmitter.
type Writer struct{}

// filteredDoc is the on-disk shape of <profile>.filtered.toml: a single file
// with the same table names the per-file loaders use, so it round-trips back
// into domain types without semantic loss.
type filteredDoc struct {
	Header     domain.Header     `toml:"header"`
	Jobs       []domain.Job      `toml:"jobs"`
	Projects   []domain.Project  `toml:"projects"`
	Edu        []domain.Edu      `toml:"edu"`
	Categories []domain.SkillCat `toml:"categories"`
}

// Emit writes the Markdown and/or filtered-TOML siblings of pdfPath, according
// to the emit flags in cfg.Render.
func (Writer) Emit(_ context.Context, pdfPath string, data domain.ResumeData, profile domain.Profile, cfg domain.Config) error {
	base := strings.TrimSuffix(pdfPath, ".pdf")

	if cfg.Render.EmitMarkdown {
		md := domain.RenderMarkdown(data, profile)
		if err := writePrivate(base+".md", md); err != nil {
			return fmt.Errorf("write markdown: %w", err)
		}
	}

	if cfg.Render.EmitFiltered {
		v := domain.VisibleResume(data)
		doc := filteredDoc{
			Header:     v.Header,
			Jobs:       v.Jobs,
			Projects:   v.Projects,
			Edu:        v.Edu,
			Categories: v.SkillCats,
		}
		raw, err := toml.Marshal(doc)
		if err != nil {
			return fmt.Errorf("marshal filtered toml: %w", err)
		}
		if err := writePrivate(base+".filtered.toml", raw); err != nil {
			return fmt.Errorf("write filtered toml: %w", err)
		}
	}

	return nil
}
