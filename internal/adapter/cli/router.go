package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// Command is a single subcommand. Top-level commands (init, prompt,
// template, render) and nested subcommands (prompt extract, template
// extract) both implement it.
type Command interface {
	Name() string
	Synopsis() string
	Run(ctx context.Context, deps Deps, args []string) error
}

// Run is the binary entrypoint. It dispatches to a subcommand or to the
// default render command. On error it prints the message to stderr and exits
// with a status determined by exitCode (2 usage, 1 resolution/other).
func Run(deps Deps) {
	if err := dispatch(context.Background(), deps, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(exitCode(err))
	}
}

func dispatch(ctx context.Context, deps Deps, args []string) error {
	cmds := topLevelCommands()

	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return cmdRender{}.Run(ctx, deps, args)
	}

	if args[0] == "help" {
		printUsage(os.Stdout, cmds)
		return nil
	}

	for _, c := range cmds {
		if c.Name() == args[0] {
			return c.Run(ctx, deps, args[1:])
		}
	}
	return usageErr(fmt.Errorf("unknown command %q (run `resumegen help` for usage)", args[0]))
}

func topLevelCommands() []Command {
	return []Command{
		cmdRender{},
		cmdInit{},
		cmdTemplate{},
		cmdPrompt{},
		cmdApply{},
		cmdTUI{},
	}
}

func printUsage(w io.Writer, cmds []Command) {
	writeln(w, "Usage: resumegen [command] [flags]")
	writeln(w, "")
	writeln(w, "When no command is given, resumegen renders the active profile (v1.0 behavior).")
	writeln(w, "")
	writeln(w, "Commands:")
	for _, c := range cmds {
		writef(w, "  %-12s %s\n", c.Name(), c.Synopsis())
	}
	writeln(w, "")
	writeln(w, "Run `resumegen <command> --help` for command-specific flags.")
}

// writeln is a thin wrapper around fmt.Fprintln that discards the unused
// return values. Usage text writes have no recovery path: an error writing
// to stderr/stdout means the terminal is gone, which we can't help.
func writeln(w io.Writer, s string) { _, _ = fmt.Fprintln(w, s) }

func writef(w io.Writer, format string, a ...any) { _, _ = fmt.Fprintf(w, format, a...) }
