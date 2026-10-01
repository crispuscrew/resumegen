package cli

import (
	"errors"
	"flag"
	"os"
	"strings"
)

// parseFlags wraps FlagSet.Parse with the tool-wide conventions: a requested
// --help is not an error (flag has already printed the usage; exit 0), and any
// real parse failure is a usage error (exit 2). Every command's Parse call goes
// through here so help behavior and exit codes cannot drift per command.
func parseFlags(fs *flag.FlagSet, args []string) (helped bool, err error) {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return true, nil
		}
		return false, usageErr(err)
	}
	return false, nil
}

// parseFlagsInterleaved is parseFlags for subcommands that mix positionals and
// flags, returning the positionals in order.
//
// flag.Parse stops at the first non-flag argument, so anything after a
// positional is silently ignored: `prompt extract analyze-jd --path ws` used to
// treat "--path" and "ws" as extra *names* and extract into the default appdir
// instead. Re-parsing the remainder after consuming each positional makes flags
// work in any position.
//
// A free-text tail whose first word begins with "-" still needs an explicit
// "--" terminator, exactly as with plain flag.Parse.
func parseFlagsInterleaved(fs *flag.FlagSet, args []string) (positionals []string, helped bool, err error) {
	rest := args
	for {
		if helped, err := parseFlags(fs, rest); helped || err != nil {
			return nil, helped, err
		}
		if fs.NArg() == 0 {
			return positionals, false, nil
		}
		positionals = append(positionals, fs.Arg(0))
		rest = fs.Args()[1:]
	}
}

// positionalArgs validates the leading positional argument(s) of a subcommand.
// A help flag in front prints the usage line on stdout and reports helped
// (exit 0, same contract as parseFlags); a missing or flag-shaped positional
// is a usage error (exit 2). Help after the positional is handled by the
// FlagSet itself via parseFlags.
func positionalArgs(args []string, min int, usage string) (helped bool, err error) {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "-help" || args[0] == "--help") {
		writeln(os.Stdout, "usage: "+usage)
		return true, nil
	}
	if len(args) < min || strings.HasPrefix(args[0], "-") {
		return false, usageErr(errors.New("usage: " + usage))
	}
	return false, nil
}

// Exit-code discipline for machine callers (SPEC section 6a): 0 ok, 1 resolution
// error (unknown id, invalid transition, missing followup args), 2 usage error
// (bad flag/date). Commands wrap their errors with usageErr/resolutionErr; the
// router reads the code via exitCode. Errors without a wrapper default to 1.

// codedError carries an intended process exit status alongside a message.
type codedError struct {
	code int
	err  error
}

func (e *codedError) Error() string { return e.err.Error() }
func (e *codedError) Unwrap() error { return e.err }

// usageErr tags err as a usage error (exit 2): a bad flag or malformed date.
func usageErr(err error) error {
	if err == nil {
		return nil
	}
	return &codedError{code: 2, err: err}
}

// resolutionErr tags err as a resolution error (exit 1): an unknown id, an
// invalid transition, or missing required arguments. Exit 1 is already the
// default, so this mainly documents intent and keeps a wrapped code stable.
func resolutionErr(err error) error {
	if err == nil {
		return nil
	}
	return &codedError{code: 1, err: err}
}

// exitCode extracts the intended exit status from err: the wrapped code when
// present, else 1 for any other non-nil error, else 0.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var ce *codedError
	if errors.As(err, &ce) {
		return ce.code
	}
	return 1
}
