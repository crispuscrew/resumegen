package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/crispuscrew/resumegen/internal/adapter/appdir"
	"github.com/crispuscrew/resumegen/internal/adapter/promptrepo"
	"github.com/crispuscrew/resumegen/internal/usecase"
	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

type cmdPrompt struct{}

func (cmdPrompt) Name() string { return "prompt" }
func (cmdPrompt) Synopsis() string {
	return "Build ready-to-paste LLM prompts from your resume and a job description"
}

func (cmdPrompt) Run(ctx context.Context, deps Deps, args []string) error {
	if len(args) == 0 {
		return usageErr(errors.New("usage: resumegen prompt <list|show|run|extract> [args]"))
	}
	switch args[0] {
	case "list":
		return promptList(ctx, deps, args[1:])
	case "show":
		return promptShow(ctx, deps, args[1:])
	case "run":
		return promptRun(ctx, deps, args[1:])
	case "extract":
		return promptExtract(ctx, deps, args[1:])
	case "help", "-h", "--help":
		fmt.Println("Usage: resumegen prompt <subcommand> [args]")
		fmt.Println("  list              List available prompt templates")
		fmt.Println("  show <name>       Show a template and the inputs it needs")
		fmt.Println("  run <name>        Fill a template and emit it (stdout, --output, or --copy)")
		fmt.Println("  extract [name...] Copy embedded prompt(s) into <appdir>/prompts/ for editing")
		return nil
	default:
		return usageErr(fmt.Errorf("unknown prompt subcommand: %s", args[0]))
	}
}

// promptRepo builds a prompt.Repo over the embedded defaults plus the resolved
// workspace prompts/ directory (appdir shadows embedded).
func promptRepo(deps Deps, appDirPath string) (prompt.Repo, error) {
	cwd, _ := os.Getwd()
	res, err := appdir.ResolveActive(appDirPath, cwd, defaultAppDir())
	if err != nil {
		return nil, err
	}
	return promptrepo.New(deps.Skeleton, os.DirFS(res.Dir)), nil
}

func promptList(ctx context.Context, deps Deps, args []string) error {
	flags := flag.NewFlagSet("prompt list", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	jsonOut := flags.Bool("json", false, "emit a stable JSON array instead of a table")
	if helped, err := parseFlags(flags, args); helped || err != nil {
		return err
	}
	repo, err := promptRepo(deps, *appDirPath)
	if err != nil {
		return err
	}
	entries, err := repo.List(ctx)
	if err != nil {
		return fmt.Errorf("list prompts: %w", err)
	}
	if *jsonOut {
		return emitJSON(listJSON(entries))
	}
	if len(entries) == 0 {
		fmt.Println("no prompts found")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	overrides := false
	for _, e := range entries {
		name := e.Name
		if e.Overridden {
			name += " *"
			overrides = true
		}
		writef(tw, "%s\t%s\n", name, e.Description)
	}
	_ = tw.Flush()
	if overrides {
		fmt.Println("\n(* = overridden by a copy in your appdir)")
	}
	return nil
}

func promptShow(ctx context.Context, deps Deps, args []string) error {
	if helped, err := positionalArgs(args, 1, "resumegen prompt show <name> [--json]"); helped || err != nil {
		return err
	}
	name := args[0]
	flags := flag.NewFlagSet("prompt show", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory")
	jsonOut := flags.Bool("json", false, "emit the template as a stable JSON object")
	if helped, err := parseFlags(flags, args[1:]); helped || err != nil {
		return err
	}
	repo, err := promptRepo(deps, *appDirPath)
	if err != nil {
		return err
	}
	t, err := repo.Load(ctx, name)
	if err != nil {
		return err
	}
	if *jsonOut {
		return emitJSON(templateJSON(t))
	}

	fmt.Printf("%s - %s\n", t.Name, t.Description)
	if len(t.Inputs) > 0 {
		fmt.Println("\nInputs:")
		tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, key := range sortedKeys(t.Inputs) {
			spec := t.Inputs[key]
			req := "optional"
			if spec.Required {
				req = "required"
			}
			writef(tw, "  {{%s}}\t%s\t%s\t%s\n", key, spec.Source, flagHint(spec), req)
		}
		_ = tw.Flush()
	}
	fmt.Printf("\n--- template ---\n%s", t.Body)
	return nil
}

func flagHint(spec prompt.InputSpec) string {
	if spec.Flag != "" {
		return "--" + spec.Flag
	}
	return ""
}

func sortedKeys(m map[string]prompt.InputSpec) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func promptExtract(ctx context.Context, deps Deps, args []string) error {
	flags := flag.NewFlagSet("prompt extract", flag.ContinueOnError)
	appDirPath := flags.String("path", "", "specific path to application directory (default: walk-up from CWD, then ~/.config/resumegen/)")
	only, helped, err := parseFlagsInterleaved(flags, args)
	if helped || err != nil {
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

	extractor := appdir.NewSkeletonExtractor(deps.Skeleton)
	report, err := usecase.ExtractSubtree(ctx, extractor, "prompts", res.Dir, only)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return errors.New("no embedded prompts in this build")
		}
		return fmt.Errorf("extract prompts: %w", err)
	}
	if len(only) > 0 && len(report.Copied) == 0 && len(report.Skipped) == 0 {
		return fmt.Errorf("no prompt matched %v", only)
	}
	printExtractReport("prompts", report)
	return nil
}
