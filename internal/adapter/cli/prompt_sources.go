package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

func (resolver resolveCtx) flagValue(key string, spec prompt.InputSpec) string {
	if value, found := resolver.flagVals[flagName(key, spec)]; found {
		return *value
	}
	return ""
}

func (resolver resolveCtx) readDataDump(ctx context.Context, key string, spec prompt.InputSpec) (string, error) {
	cfgSource, result, err := layeredConfigSource(resolver.appdir)
	if err != nil {
		return "", err
	}
	outputDir := "output"
	config, configErr := cfgSource.Load(ctx)
	if configErr == nil && config.Paths.OutputDir != "" {
		outputDir = config.Paths.OutputDir
	}
	path := filepath.Join(result.Dir, outputDir, resolver.profile+".md")
	raw, err := os.ReadFile(path)
	if err != nil {
		if spec.Required {
			return "", fmt.Errorf("input %q: no data-dump at %s - enable [render] emit_markdown and render profile %q first", key, path, resolver.profile)
		}
		return spec.Default, nil
	}
	return string(raw), nil
}

func (resolver resolveCtx) readAppField(ctx context.Context, key string, spec prompt.InputSpec) (string, error) {
	if resolver.appID == "" {
		return resolver.requireOrDefault(key, spec, "")
	}
	tracker, err := tracker(ctx, resolver.deps, resolver.appdir)
	if err != nil {
		return "", err
	}
	application, err := tracker.Get(ctx, resolver.appID)
	if err != nil {
		return "", fmt.Errorf("input %q: %w", key, err)
	}
	switch spec.Field {
	case "company":
		return resolver.requireOrDefault(key, spec, application.Company)
	case "role":
		return resolver.requireOrDefault(key, spec, application.Role)
	case "status":
		return resolver.requireOrDefault(key, spec, string(application.Status))
	case "source":
		return resolver.requireOrDefault(key, spec, application.Source)
	case "notes":
		return resolver.requireOrDefault(key, spec, application.Notes)
	case "jd":
		if application.JDPath == "" {
			return resolver.requireOrDefault(key, spec, "")
		}
		raw, err := os.ReadFile(application.JDPath)
		if err != nil {
			return "", fmt.Errorf("input %q: read jd_path %s: %w", key, application.JDPath, err)
		}
		return string(raw), nil
	default:
		return "", fmt.Errorf("input %q: unknown app-id field %q", key, spec.Field)
	}
}

func (resolver resolveCtx) readInteractive(key string, spec prompt.InputSpec) (string, error) {
	if resolver.noInput {
		if spec.Default != "" {
			return spec.Default, nil
		}
		if spec.Required {
			return "", fmt.Errorf("input %q needs interactive entry but --no-input is set", key)
		}
		return "", nil
	}
	fmt.Fprintf(os.Stderr, "%s: ", key)
	line, err := resolver.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("input %q: read: %w", key, err)
	}
	return resolver.requireOrDefault(key, spec, strings.TrimRight(line, "\n"))
}

func (resolver resolveCtx) requireOrDefault(key string, spec prompt.InputSpec, value string) (string, error) {
	if value != "" {
		return value, nil
	}
	if spec.Default != "" {
		return spec.Default, nil
	}
	if spec.Required {
		return "", fmt.Errorf("required input %q is unset (%s)", key, sourceHint(spec))
	}
	return "", nil
}
