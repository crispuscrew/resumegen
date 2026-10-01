package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

type resolveCtx struct {
	deps     Deps
	appdir   string
	profile  string
	appID    string
	noInput  bool
	flagVals map[string]*string
	stdin    io.Reader
	reader   *bufio.Reader
}

func (resolver resolveCtx) resolve(ctx context.Context, key string, spec prompt.InputSpec) (string, error) {
	switch spec.Source {
	case prompt.SourceFlag:
		value := resolver.flagValue(key, spec)
		if value == "" {
			if fallback, found, err := resolver.appFallback(ctx, key, spec); err != nil {
				return "", err
			} else if found {
				return fallback, nil
			}
		}
		return resolver.requireOrDefault(key, spec, value)
	case prompt.SourceFile, prompt.SourceJDFile:
		path := resolver.flagValue(key, spec)
		if path == "" {
			if fallback, found, err := resolver.appFallback(ctx, key, spec); err != nil {
				return "", err
			} else if found {
				return fallback, nil
			}
			return resolver.requireOrDefault(key, spec, "")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("input %q: read %s: %w", key, path, err)
		}
		if spec.Source == prompt.SourceFile {
			return resolver.requireOrDefault(key, spec, string(raw))
		}
		return string(raw), nil
	case prompt.SourceDataDump:
		return resolver.readDataDump(ctx, key, spec)
	case prompt.SourceAppID:
		return resolver.readAppField(ctx, key, spec)
	case prompt.SourceStdin:
		if resolver.noInput && isTerminal(resolver.stdin) {
			return resolver.requireOrDefault(key, spec, "")
		}
		if resolver.noInput && !resolver.stdinHasData() {
			return resolver.requireOrDefault(key, spec, "")
		}
		raw, err := io.ReadAll(resolver.reader)
		if err != nil {
			return "", fmt.Errorf("input %q: read stdin: %w", key, err)
		}
		return resolver.requireOrDefault(key, spec, strings.TrimRight(string(raw), "\n"))
	case prompt.SourcePrompt:
		return resolver.readInteractive(key, spec)
	default:
		return "", fmt.Errorf("input %q: unsupported source %q", key, spec.Source)
	}
}

// TUI values use the same resolution rules without reading os.Stdin.
func (resolver resolveCtx) resolveFromValues(ctx context.Context, template prompt.PromptTemplate, values map[string]string) (prompt.PromptInput, error) {
	inputs := prompt.PromptInput{}
	for key, spec := range template.Inputs {
		var value string
		var err error
		switch spec.Source {
		case prompt.SourceFlag:
			value = strings.TrimSpace(values[key])
			if value == "" {
				if fallback, found, ferr := resolver.appFallback(ctx, key, spec); ferr != nil {
					return nil, ferr
				} else if found {
					inputs[key] = fallback
					continue
				}
			}
			value, err = resolver.requireOrDefault(key, spec, value)
		case prompt.SourcePrompt, prompt.SourceStdin:
			value, err = resolver.requireOrDefault(key, spec, strings.TrimSpace(values[key]))
		case prompt.SourceFile, prompt.SourceJDFile:
			path := strings.TrimSpace(values[key])
			if path == "" {
				if fallback, found, ferr := resolver.appFallback(ctx, key, spec); ferr != nil {
					return nil, ferr
				} else if found {
					inputs[key] = fallback
					continue
				}
				value, err = resolver.requireOrDefault(key, spec, "")
				break
			}
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil, fmt.Errorf("input %q: read %s: %w", key, path, readErr)
			}
			value = string(raw)
			if spec.Source == prompt.SourceFile {
				value, err = resolver.requireOrDefault(key, spec, value)
			}
		case prompt.SourceDataDump:
			value, err = resolver.readDataDump(ctx, key, spec)
		case prompt.SourceAppID:
			value, err = resolver.readAppField(ctx, key, spec)
		default:
			err = fmt.Errorf("input %q: unsupported source %q", key, spec.Source)
		}
		if err != nil {
			return nil, err
		}
		inputs[key] = value
	}
	return inputs, nil
}
