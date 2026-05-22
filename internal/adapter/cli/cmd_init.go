package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

type cmdInit struct{}

func (cmdInit) Name() string     { return "init" }
func (cmdInit) Synopsis() string { return "Bootstrap a workspace (marker + example data + profiles)" }

func (cmdInit) Run(ctx context.Context, deps Deps, args []string) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	var (
		bare        = flags.Bool("bare", false, "create only the workspace marker; skip example data and profiles")
		withExample = flags.Bool("with-example", false, "(default) copy example data and profiles")
		fullExample = flags.Bool("full-example", false, "also extract templates and prompts for editing")
		name        = flags.String("name", "", "workspace name written to the marker (default: basename of the target dir)")
		description = flags.String("description", "", "workspace description written to the marker")
	)
	flags.Usage = func() {
		out := flags.Output()
		writeln(out, "Usage: resumegen init [flags] [path]")
		writeln(out, "  Creates .resumegen/workspace.toml plus optional example content.")
		writeln(out, "  Default path: current directory. Flags must come before the path.")
		writeln(out, "")
		writeln(out, "Flags:")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *bare && *fullExample {
		return errors.New("--bare and --full-example are mutually exclusive")
	}
	if *bare && *withExample {
		return errors.New("--bare and --with-example are mutually exclusive")
	}

	var target string
	switch flags.NArg() {
	case 0:
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cwd: %w", err)
		}
		target = cwd
	case 1:
		target = flags.Arg(0)
	default:
		return fmt.Errorf("init takes at most one path argument; got %d", flags.NArg())
	}
	target, err := appdir.ExpandAbs(target)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(target, 0o755); err != nil {
		return fmt.Errorf("mkdir target: %w", err)
	}

	wsName := *name
	if wsName == "" {
		wsName = filepath.Base(target)
	}

	repo := appdir.NewWorkspaceRepo()
	markerCreated := true
	saveErr := repo.Save(ctx, target, domain.Workspace{
		SchemaVersion: domain.CurrentWorkspaceSchema,
		Workspace: domain.WorkspaceMeta{
			Name:        wsName,
			Description: *description,
		},
	})
	if saveErr != nil {
		if !errors.Is(saveErr, appdir.ErrWorkspaceExists) {
			return saveErr
		}
		markerCreated = false
	}

	markerPath := filepath.Join(target, appdir.MarkerSubpath)
	if markerCreated {
		fmt.Printf("Created workspace marker: %s\n", markerPath)
	} else {
		fmt.Printf("Workspace marker already present at %s (left unchanged)\n", markerPath)
		if *name != "" || *description != "" {
			fmt.Fprintf(os.Stderr, "warning: --name/--description ignored because marker already exists; edit %s by hand to change it\n", markerPath)
		}
	}

	if *bare {
		return nil
	}

	subtrees := []string{"data", "profiles"}
	if *fullExample {
		subtrees = append(subtrees, "templates", "prompts")
	}

	extractor := appdir.NewSkeletonExtractor(deps.Skeleton)
	for _, sub := range subtrees {
		report, err := usecase.ExtractSubtree(ctx, extractor, sub, target, nil)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				fmt.Printf("Skipped subtree %q (not present in this build).\n", sub)
				continue
			}
			return fmt.Errorf("extract %s: %w", sub, err)
		}
		printExtractReport(sub, report)
	}

	return nil
}

func printExtractReport(label string, report usecase.ExtractReport) {
	if len(report.Copied) == 0 && len(report.Skipped) == 0 {
		fmt.Printf("[%s] no files in skeleton subtree\n", label)
		return
	}
	for _, p := range report.Copied {
		fmt.Printf("  copied  %s\n", p)
	}
	for _, p := range report.Skipped {
		fmt.Printf("  skipped %s (already exists)\n", p)
	}
}
