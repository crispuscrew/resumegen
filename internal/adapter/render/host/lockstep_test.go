package host_test

import (
	"bytes"
	"context"
	"io/fs"
	"testing"

	"github.com/crispuscrew/resumegen"
	"github.com/crispuscrew/resumegen/internal/adapter/render/host"
	"github.com/crispuscrew/resumegen/internal/adapter/render/sanitize"
	"github.com/crispuscrew/resumegen/internal/adapter/tomlrepo"
	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

// BuildTypstSource filters by Reason == Included internally; VisibleResume is
// the single source of truth the .md/.filtered.toml emitters project from. If
// the two predicates ever diverge, the emitted siblings would stop matching
// the PDF. This test pins them together: pre-filtering with VisibleResume must
// be a no-op for the generated typst source, across both shipped profiles and
// after a trim pass.
func TestBuildTypstSource_LockstepWithVisibleResume(t *testing.T) {
	skeleton, err := fs.Sub(resumegen.Defaults, "defaultAppDir")
	if err != nil {
		t.Fatalf("sub-fs: %v", err)
	}
	ctx := context.Background()
	cfg, err := tomlrepo.NewConfigSource(skeleton).Load(ctx)
	if err != nil {
		t.Fatalf("config: %v", err)
	}

	for _, name := range []string{"default", "robotics-uav"} {
		profile, err := tomlrepo.NewProfileRepo(skeleton).Load(ctx, name)
		if err != nil {
			t.Fatalf("profile %s: %v", name, err)
		}
		data, err := tomlrepo.NewResumeRepo(skeleton).Load(ctx)
		if err != nil {
			t.Fatalf("data: %v", err)
		}
		data = usecase.Score(data, profile.Tags, cfg.Score)
		if trimmed, ok := usecase.TrimLowest(data, cfg.Render.MinElements); ok {
			data = trimmed // cover Trimmed, not just Filtered
		}

		direct, err := host.BuildTypstSource(data, profile, sanitize.Strict)
		if err != nil {
			t.Fatalf("%s: build direct: %v", name, err)
		}
		prefiltered, err := host.BuildTypstSource(domain.VisibleResume(data), profile, sanitize.Strict)
		if err != nil {
			t.Fatalf("%s: build prefiltered: %v", name, err)
		}
		if !bytes.Equal(direct, prefiltered) {
			t.Errorf("%s: VisibleResume and BuildTypstSource disagree on visibility — emitted files would not match the PDF", name)
		}
	}
}
