package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/crispuscrew/resumegen/internal/adapter/container"
	"github.com/crispuscrew/resumegen/internal/adapter/render/host"
	"github.com/crispuscrew/resumegen/internal/domain"
)

// renderRouter implements usecase.Renderer by dispatching on
// cfg.Render.UseContainer. In "off" mode it is byte-identical to the v1.0
// host renderer (no banner, no extra side-effects).
type renderRouter struct {
	appdir    string
	engine    container.Engine
	engineOK  bool
	image     string
	cfile     []byte
	runner    container.Runner
	bannerOut io.Writer // typically os.Stderr; banner is only printed in container modes
}

func (r renderRouter) Render(ctx context.Context, data domain.ResumeData, profile domain.Profile, cfg domain.Config) (string, float64, error) {
	mode, err := domain.ParseContainerMode(cfg.Render.UseContainer)
	if err != nil {
		return "", 0, err
	}
	switch mode {
	case domain.ContainerOff:
		return r.renderHost(ctx, data, profile, cfg)
	case domain.ContainerOn:
		if !r.engineOK {
			return "", 0, errors.New("render.use_container=\"true\" but no container engine (podman/docker) found on PATH")
		}
		if err := r.ensureImage(ctx); err != nil {
			return "", 0, err
		}
		r.banner("container", r.engine.Name)
		return r.renderContainer(ctx, data, profile, cfg)
	case domain.ContainerAuto:
		if !r.engineOK {
			r.banner("host", "no container engine on PATH")
			return r.renderHost(ctx, data, profile, cfg)
		}
		if err := r.ensureImage(ctx); err != nil {
			r.banner("host", fmt.Sprintf("image build failed: %v", err))
			return r.renderHost(ctx, data, profile, cfg)
		}
		r.banner("container", r.engine.Name)
		return r.renderContainer(ctx, data, profile, cfg)
	}
	return "", 0, fmt.Errorf("unreachable container mode %v", mode)
}

func (r renderRouter) renderHost(ctx context.Context, data domain.ResumeData, profile domain.Profile, cfg domain.Config) (string, float64, error) {
	return host.Renderer{Appdir: r.appdir}.Render(ctx, data, profile, cfg)
}

func (r renderRouter) renderContainer(ctx context.Context, data domain.ResumeData, profile domain.Profile, cfg domain.Config) (string, float64, error) {
	cr := container.NewRenderer(r.engine, r.image, r.appdir)
	cr.Runner = r.runner
	return cr.Render(ctx, data, profile, cfg)
}

func (r renderRouter) ensureImage(ctx context.Context) error {
	if len(r.cfile) == 0 {
		return errors.New("no embedded Containerfile (build issue)")
	}
	return container.EnsureImage(ctx, r.runner, r.engine, r.image, r.cfile, r.bannerOut, r.bannerOut)
}

func (r renderRouter) banner(mode, detail string) {
	if r.bannerOut == nil {
		return
	}
	_, _ = fmt.Fprintf(r.bannerOut, "rendering: %s (%s)\n", mode, detail)
}
