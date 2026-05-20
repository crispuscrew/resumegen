package usecase_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

type fakeConfig struct {
	cfg domain.Config
	err error
}

func (f *fakeConfig) Load(_ context.Context) (domain.Config, error) { return f.cfg, f.err }

type fakeProfile struct {
	p   domain.Profile
	err error
}

func (f *fakeProfile) Load(_ context.Context, _ string) (domain.Profile, error) { return f.p, f.err }

type fakeResume struct {
	d   domain.ResumeData
	err error
}

func (f *fakeResume) Load(_ context.Context) (domain.ResumeData, error) { return f.d, f.err }

type fakeRenderer struct {
	pages    []float64
	out      string
	err      error
	calls    int
	lastProf domain.Profile
}

func (f *fakeRenderer) Render(_ context.Context, _ domain.ResumeData, p domain.Profile, _ domain.Config) (string, float64, error) {
	if f.err != nil {
		return "", 0, f.err
	}
	f.lastProf = p
	pg := f.pages[0]
	if len(f.pages) > 1 {
		f.pages = f.pages[1:]
	}
	f.calls++
	return f.out, pg, nil
}

type fakeBootstrap struct {
	called int
	err    error
	onCall func()
}

func (f *fakeBootstrap) EnsureWorkspace(_ context.Context, _ string) error {
	f.called++
	if f.onCall != nil {
		f.onCall()
	}
	return f.err
}

func TestGenerate_HappyPath(t *testing.T) {
	r := &fakeRenderer{pages: []float64{0.5}, out: "/tmp/x.pdf"}
	g := usecase.Generator{
		Config:    &fakeConfig{cfg: domain.Config{Render: domain.Render{PageLimit: 1.0}}},
		Profiles:  &fakeProfile{p: domain.Profile{Lang: "en"}},
		Resumes:   &fakeResume{},
		Renderer:  r,
		Bootstrap: &fakeBootstrap{},
	}
	out, err := g.Generate(context.Background(), usecase.GenerateInput{ProfileName: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "/tmp/x.pdf" {
		t.Errorf("got %q", out)
	}
	if r.calls != 1 {
		t.Errorf("renderer called %d times, want 1", r.calls)
	}
}

func TestGenerate_LangOverrideReachesRenderer(t *testing.T) {
	r := &fakeRenderer{pages: []float64{0.1}, out: "out.pdf"}
	g := usecase.Generator{
		Config:    &fakeConfig{cfg: domain.Config{Render: domain.Render{PageLimit: 1.0}}},
		Profiles:  &fakeProfile{p: domain.Profile{Lang: "en"}},
		Resumes:   &fakeResume{},
		Renderer:  r,
		Bootstrap: &fakeBootstrap{},
	}
	if _, err := g.Generate(context.Background(), usecase.GenerateInput{ProfileName: "default", LangOverride: "ru"}); err != nil {
		t.Fatal(err)
	}
	if r.lastProf.Lang != "ru" {
		t.Errorf("renderer saw lang=%q, want ru", r.lastProf.Lang)
	}
}

func TestGenerate_BootstrapFiresOnMissingThenRetries(t *testing.T) {
	cfg := &fakeConfig{err: fmt.Errorf("%w: config.toml", usecase.ErrWorkspaceMissing)}
	bs := &fakeBootstrap{onCall: func() { cfg.err = nil }}
	g := usecase.Generator{
		Config:    cfg,
		Profiles:  &fakeProfile{},
		Resumes:   &fakeResume{},
		Renderer:  &fakeRenderer{pages: []float64{0.1}, out: "x.pdf"},
		Bootstrap: bs,
	}
	if _, err := g.Generate(context.Background(), usecase.GenerateInput{}); err != nil {
		t.Fatal(err)
	}
	if bs.called != 1 {
		t.Errorf("bootstrap called %d times, want 1", bs.called)
	}
}

func TestGenerate_BootstrapDeclineFails(t *testing.T) {
	cfg := &fakeConfig{err: fmt.Errorf("%w: config.toml", usecase.ErrWorkspaceMissing)}
	bs := &fakeBootstrap{err: errors.New("user declined")}
	g := usecase.Generator{
		Config:    cfg,
		Profiles:  &fakeProfile{},
		Resumes:   &fakeResume{},
		Renderer:  &fakeRenderer{},
		Bootstrap: bs,
	}
	_, err := g.Generate(context.Background(), usecase.GenerateInput{})
	if err == nil {
		t.Fatal("expected error when bootstrap returns error")
	}
}

func TestGenerate_CannotFitPage(t *testing.T) {
	// PageLimit=1, renderer always returns >1 page. Generator should exhaust
	// trim attempts (no trimmable data) and return ErrCannotFitPage.
	g := usecase.Generator{
		Config:    &fakeConfig{cfg: domain.Config{Render: domain.Render{PageLimit: 1.0}}},
		Profiles:  &fakeProfile{},
		Resumes:   &fakeResume{},
		Renderer:  &fakeRenderer{pages: []float64{2.0}, out: "x.pdf"},
		Bootstrap: &fakeBootstrap{},
	}
	_, err := g.Generate(context.Background(), usecase.GenerateInput{})
	if !errors.Is(err, usecase.ErrCannotFitPage) {
		t.Errorf("got %v, want ErrCannotFitPage", err)
	}
}

func TestGenerate_PropagatesNonMissingConfigError(t *testing.T) {
	g := usecase.Generator{
		Config:    &fakeConfig{err: errors.New("bad TOML")},
		Profiles:  &fakeProfile{},
		Resumes:   &fakeResume{},
		Renderer:  &fakeRenderer{},
		Bootstrap: &fakeBootstrap{},
	}
	if _, err := g.Generate(context.Background(), usecase.GenerateInput{}); err == nil {
		t.Fatal("expected error to bubble up")
	}
}
