package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

type cmdPrompt struct{}

func (cmdPrompt) Name() string     { return "prompt" }
func (cmdPrompt) Synopsis() string { return "Manage LLM prompt templates (library lands in v1.3; extract plumbing only in v1.1)" }

func (cmdPrompt) Run(ctx context.Context, deps Deps, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: resumegen prompt extract <name> [--path <dir>]")
	}
	switch args[0] {
	case "extract":
		return promptExtract(ctx, deps, args[1:])
	case "help", "-h", "--help":
		fmt.Println("Usage: resumegen prompt <subcommand> [args]")
		fmt.Println("  extract <name>   Copy an embedded prompt into <appdir>/prompts/")
		fmt.Println()
		fmt.Println("Note: the prompt library is shipping in v1.3. The v1.1 binary contains")
		fmt.Println("no embedded prompts, so `extract` will report \"no prompts in this build.\"")
		return nil
	default:
		return fmt.Errorf("unknown prompt subcommand: %s", args[0])
	}
}

func promptExtract(ctx context.Context, deps Deps, args []string) error {
	flags := flag.NewFlagSet("prompt extract", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory (default: walk-up from CWD, then ~/.config/resumegen/)")
	if err := flags.Parse(args); err != nil {
		return err
	}

	cwd, _ := os.Getwd()
	res, err := appdir.ResolveActive(*appDirPath, cwd, defaultAppDir())
	if err != nil {
		return err
	}

	if err := os.MkdirAll(res.Dir, 0o755); err != nil {
		return fmt.Errorf("mkdir appdir: %w", err)
	}

	only := flags.Args()
	extractor := appdir.NewSkeletonExtractor(deps.Skeleton)
	report, err := usecase.ExtractSubtree(ctx, extractor, "prompts", res.Dir, only)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errors.New("no embedded prompts in this build (prompt library ships in v1.3)")
		}
		return fmt.Errorf("extract prompts: %w", err)
	}
	if len(only) > 0 && len(report.Copied) == 0 && len(report.Skipped) == 0 {
		return fmt.Errorf("no prompt matched %v", only)
	}
	printExtractReport("prompts", report)
	return nil
}
