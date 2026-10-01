package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

// Only JD-file inputs and company/role flags inherit tracked application data.
// Generic file inputs must always use their own path or explicit default.
func (resolver resolveCtx) appFallback(ctx context.Context, key string, spec prompt.InputSpec) (string, bool, error) {
	if resolver.appID == "" {
		return "", false, nil
	}
	isJD := spec.Source == prompt.SourceJDFile
	name := flagName(key, spec)
	isField := spec.Source == prompt.SourceFlag && (name == "company" || name == "role")
	if !isJD && !isField {
		return "", false, nil
	}
	tracker, err := tracker(ctx, resolver.deps, resolver.appdir)
	if err != nil {
		return "", false, err
	}
	application, err := tracker.Get(ctx, resolver.appID)
	if err != nil {
		return "", false, fmt.Errorf("--app: %w", err)
	}
	if isJD {
		if application.JDPath == "" {
			return "", false, nil
		}
		raw, err := os.ReadFile(application.JDPath)
		if err != nil {
			return "", false, fmt.Errorf("input %q: read jd_path %s: %w", key, application.JDPath, err)
		}
		return string(raw), true, nil
	}
	if name == "company" {
		return application.Company, application.Company != "", nil
	}
	return application.Role, application.Role != "", nil
}
