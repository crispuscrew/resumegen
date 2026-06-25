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

type cmdTemplate struct{}

func (cmdTemplate) Name() string     { return "template" }
func (cmdTemplate) Synopsis() string { return "Manage Typst templates (extract embedded defaults)" }

func (cmdTemplate) Run(ctx context.Context, deps Deps, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: resumegen template extract [name...] [--path <dir>]")
	}
	switch args[0] {
	case "extract":
		return templateExtract(ctx, deps, args[1:])
	case "help", "-h", "--help":
		fmt.Println("Usage: resumegen template <subcommand> [args]")
		fmt.Println("  extract [name...]   Copy embedded template(s) into <appdir>/templates/")
		return nil
	default:
		return fmt.Errorf("unknown template subcommand: %s", args[0])
	}
}

func templateExtract(ctx context.Context, deps Deps, args []string) error {
	flags := flag.NewFlagSet("template extract", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory (default: walk-up from CWD, then ~/.config/resumegen/)")
	flags.Usage = func() {
		out := flags.Output()
		writeln(out, "Usage: resumegen template extract [name...] [--path <dir>]")
		writeln(out, "  Copies embedded Typst template(s) into <appdir>/templates/.")
		writeln(out, "  With no name, extracts every template. Names match basename with or without extension.")
		writeln(out, "  Existing files are never overwritten.")
		writeln(out, "")
		writeln(out, "Flags:")
		flags.PrintDefaults()
	}
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
	report, err := usecase.ExtractSubtree(ctx, extractor, "templates", res.Dir, only)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errors.New("no templates embedded in this build")
		}
		return fmt.Errorf("extract templates: %w", err)
	}
	if len(only) > 0 && len(report.Copied) == 0 && len(report.Skipped) == 0 {
		return fmt.Errorf("no template matched %v", only)
	}
	printExtractReport("templates", report)
	return nil
}
