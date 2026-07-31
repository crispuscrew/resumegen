package emit_test

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/crispuscrew/resumegen"
	"github.com/crispuscrew/resumegen/internal/adapter/render/emit"
	"github.com/crispuscrew/resumegen/internal/adapter/tomlrepo"
	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

func scored() domain.ResumeData {
	return domain.ResumeData{
		Header: domain.Header{Name: domain.I18n{"en": "Ada"}},
		Jobs: []domain.Job{
			{
				Meta:    domain.Meta{Tags: []string{"go"}, Reason: domain.Included},
				Title:   domain.I18n{"en": "Engineer"},
				Company: domain.I18n{"en": "Acme"},
				Bullets: []domain.Bullet{
					{Meta: domain.Meta{Reason: domain.Included}, Text: domain.I18n{"en": "Built *X*"}},
					{Meta: domain.Meta{Reason: domain.Trimmed}, Text: domain.I18n{"en": "dropped"}},
				},
			},
			{Meta: domain.Meta{Reason: domain.Filtered}, Company: domain.I18n{"en": "Hidden"}},
		},
		Edu: []domain.Edu{{Title: domain.I18n{"en": "MIT"}}},
	}
}

func TestEmit_WritesEnabledFiles(t *testing.T) {
	dir := t.TempDir()
	pdf := filepath.Join(dir, "go-backend.pdf")

	cfg := domain.Config{}
	cfg.Render.EmitMarkdown = true
	cfg.Render.EmitFiltered = true

	if err := (emit.Writer{}).Emit(context.Background(), pdf, scored(), domain.Profile{Lang: "en"}, cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "go-backend.md")); err != nil {
		t.Errorf("markdown not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "go-backend.filtered.toml")); err != nil {
		t.Errorf("filtered toml not written: %v", err)
	}
}

func TestEmit_FlagsOffWriteNothing(t *testing.T) {
	dir := t.TempDir()
	pdf := filepath.Join(dir, "x.pdf")

	if err := (emit.Writer{}).Emit(context.Background(), pdf, scored(), domain.Profile{Lang: "en"}, domain.Config{}); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("flags off should write no files, got %d", len(entries))
	}
}

// The filtered TOML must reload into domain types with only the visible
// entities and no semantic loss.
func TestEmit_FilteredTOMLRoundTrips(t *testing.T) {
	dir := t.TempDir()
	pdf := filepath.Join(dir, "x.pdf")
	cfg := domain.Config{}
	cfg.Render.EmitFiltered = true

	if err := (emit.Writer{}).Emit(context.Background(), pdf, scored(), domain.Profile{Lang: "en"}, cfg); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "x.filtered.toml"))
	if err != nil {
		t.Fatal(err)
	}

	var doc struct {
		Header   domain.Header    `toml:"header"`
		Jobs     []domain.Job     `toml:"jobs"`
		Edu      []domain.Edu     `toml:"edu"`
		Projects []domain.Project `toml:"projects"`
	}
	if err := toml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("filtered toml does not reload: %v", err)
	}

	if len(doc.Jobs) != 1 {
		t.Fatalf("want 1 visible job, got %d", len(doc.Jobs))
	}
	if doc.Jobs[0].Company.Lang("en") != "Acme" {
		t.Errorf("job company lost in round-trip: %+v", doc.Jobs[0])
	}
	if len(doc.Jobs[0].Bullets) != 1 || doc.Jobs[0].Bullets[0].Text.Lang("en") != "Built *X*" {
		t.Errorf("want only the included bullet with raw markup, got %+v", doc.Jobs[0].Bullets)
	}
	if doc.Header.Name.Lang("en") != "Ada" {
		t.Errorf("header lost in round-trip")
	}
	if len(doc.Edu) != 1 {
		t.Errorf("education lost in round-trip")
	}
}

// Same table names and field order as emit's on-disk shape, so a re-marshal
// after reload is byte-comparable to the emitted file.
type filteredShape struct {
	Header     domain.Header     `toml:"header"`
	Jobs       []domain.Job      `toml:"jobs"`
	Projects   []domain.Project  `toml:"projects"`
	Edu        []domain.Edu      `toml:"edu"`
	Categories []domain.SkillCat `toml:"categories"`
}

// Runs the real embedded example data through score + emit, then checks the
// filtered TOML is a fixed point: reload -> re-marshal reproduces the file
// byte-for-byte, i.e. nothing the format can express was lost. Also pins that
// computed Meta fields (Score, Reason) never leak into the output.
func TestEmit_FilteredTOML_CorpusFixedPoint(t *testing.T) {
	skeleton, err := fs.Sub(resumegen.Defaults, "defaultAppDir")
	if err != nil {
		t.Fatalf("sub-fs: %v", err)
	}
	ctx := context.Background()
	cfg, err := tomlrepo.NewConfigSource(skeleton).Load(ctx)
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	profile, err := tomlrepo.NewProfileRepo(skeleton).Load(ctx, "default")
	if err != nil {
		t.Fatalf("profile: %v", err)
	}
	data, err := tomlrepo.NewResumeRepo(skeleton).Load(ctx)
	if err != nil {
		t.Fatalf("data: %v", err)
	}
	data = usecase.Score(data, profile.Tags, cfg.Score)

	dir := t.TempDir()
	cfg.Render.EmitFiltered = true
	if err := (emit.Writer{}).Emit(ctx, filepath.Join(dir, "default.pdf"), data, profile, cfg); err != nil {
		t.Fatal(err)
	}
	emitted, err := os.ReadFile(filepath.Join(dir, "default.filtered.toml"))
	if err != nil {
		t.Fatal(err)
	}

	for _, leak := range []string{"Score =", "Reason ="} {
		if bytes.Contains(emitted, []byte(leak)) {
			t.Errorf("computed field leaked into filtered toml: %s", leak)
		}
	}

	var doc filteredShape
	if err := toml.Unmarshal(emitted, &doc); err != nil {
		t.Fatalf("filtered toml does not reload: %v", err)
	}
	remarshaled, err := toml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(emitted, remarshaled) {
		t.Errorf("filtered toml is not a fixed point under reload+marshal - semantic loss:\n--- emitted ---\n%s\n--- remarshaled ---\n%s", emitted, remarshaled)
	}

	v := domain.VisibleResume(data)
	if len(doc.Jobs) != len(v.Jobs) || len(doc.Projects) != len(v.Projects) || len(doc.Categories) != len(v.SkillCats) || len(doc.Edu) != len(v.Edu) {
		t.Errorf("reloaded counts (jobs=%d projects=%d cats=%d edu=%d) differ from visible projection (%d, %d, %d, %d)",
			len(doc.Jobs), len(doc.Projects), len(doc.Categories), len(doc.Edu),
			len(v.Jobs), len(v.Projects), len(v.SkillCats), len(v.Edu))
	}
}
