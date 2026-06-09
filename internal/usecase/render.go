package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// Generator orchestrates the load → score → render → trim pipeline.
type Generator struct {
	Config        ConfigSource
	Profiles      ProfileRepo
	Resumes       ResumeRepo
	Renderer      Renderer
	PostProcessor PDFPostProcessor // optional; nil skips post-processing
	Bootstrap     Bootstrap
}

// GenerateInput captures user-supplied parameters for one render run.
type GenerateInput struct {
	ProfileName  string
	LangOverride string // "" leaves the profile's lang untouched
	ForceUnsafe  bool   // OR-combined with cfg.Render.ForceUnsafe
}

// Generate runs the full pipeline and returns the absolute path to the rendered PDF.
func (g *Generator) Generate(ctx context.Context, in GenerateInput) (string, error) {
	cfg, profile, data, err := g.loadAll(ctx, in.ProfileName)
	if err != nil {
		return "", err
	}
	if in.LangOverride != "" {
		profile.Lang = in.LangOverride
	}
	cfg.Render.ForceUnsafe = cfg.Render.ForceUnsafe || in.ForceUnsafe

	if err := domain.ValidateInput(data, cfg.Render.StrictInput, cfg.Render.Limits); err != nil {
		return "", fmt.Errorf("input validation: %w", err)
	}

	data = Score(data, profile.Tags, cfg.Score)

	for {
		outPath, pages, err := g.Renderer.Render(ctx, data, profile, cfg)
		if err != nil {
			return "", fmt.Errorf("render: %w", err)
		}
		need, err := TrimIsNeeded(pages, cfg.Render.PageLimit)
		if err != nil {
			return "", err
		}
		if !need {
			if cfg.Render.StripMetadata && g.PostProcessor != nil {
				if err := g.PostProcessor.Strip(ctx, outPath); err != nil {
					return "", fmt.Errorf("strip metadata: %w", err)
				}
			}
			return outPath, nil
		}
		var trimmed bool
		data, trimmed = TrimLowest(data, cfg.Render.MinElements)
		if !trimmed {
			return "", ErrCannotFitPage
		}
	}
}

func (g *Generator) loadAll(ctx context.Context, profileName string) (domain.Config, domain.Profile, domain.ResumeData, error) {
	for {
		cfg, cerr := g.Config.Load(ctx)
		if errors.Is(cerr, ErrWorkspaceMissing) {
			if berr := g.Bootstrap.EnsureWorkspace(ctx, "config.toml"); berr != nil {
				return domain.Config{}, domain.Profile{}, domain.ResumeData{}, berr
			}
			continue
		}
		if cerr != nil {
			return domain.Config{}, domain.Profile{}, domain.ResumeData{}, fmt.Errorf("load config: %w", cerr)
		}

		profile, perr := g.Profiles.Load(ctx, profileName)
		if errors.Is(perr, ErrWorkspaceMissing) {
			if berr := g.Bootstrap.EnsureWorkspace(ctx, "profile "+profileName); berr != nil {
				return domain.Config{}, domain.Profile{}, domain.ResumeData{}, berr
			}
			continue
		}
		if perr != nil {
			return domain.Config{}, domain.Profile{}, domain.ResumeData{}, fmt.Errorf("load profile %q: %w", profileName, perr)
		}

		data, derr := g.Resumes.Load(ctx)
		if errors.Is(derr, ErrWorkspaceMissing) {
			if berr := g.Bootstrap.EnsureWorkspace(ctx, "data"); berr != nil {
				return domain.Config{}, domain.Profile{}, domain.ResumeData{}, berr
			}
			continue
		}
		if derr != nil {
			return domain.Config{}, domain.Profile{}, domain.ResumeData{}, fmt.Errorf("load data: %w", derr)
		}

		return cfg, profile, data, nil
	}
}
