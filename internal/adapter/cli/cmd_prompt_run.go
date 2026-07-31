package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/crispuscrew/resumegen/internal/adapter/clipboard"
	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

// reservedRunFlags are the built-in `prompt run` flags; a template input may not
// claim one of these names, or registering it would collide (previously a panic).
var reservedRunFlags = map[string]bool{
	"path": true, "profile": true, "output": true,
	"copy": true, "no-input": true, "json": true, "app": true,
}

// promptRun fills a template's inputs and emits the result to the chosen sink.
func promptRun(ctx context.Context, deps Deps, args []string) error {
	if helped, err := positionalArgs(args, 1, "resumegen prompt run <name> [flags]"); helped || err != nil {
		return err
	}
	name := args[0]
	rest := args[1:]

	// Pre-scan --path so we can load the template before knowing its dynamic flags.
	repo, err := promptRepo(deps, scanFlag(rest, "path"))
	if err != nil {
		return err
	}
	t, err := repo.Load(ctx, name)
	if err != nil {
		return err
	}

	flags := flag.NewFlagSet("prompt run", flag.ContinueOnError)
	var (
		appDirPath = flags.String("path", "", "specific path to application directory")
		profile    = flags.String("profile", "default", "profile whose data-dump to read")
		output     = flags.String("output", "", "write the prompt to this file instead of stdout")
		copyOut    = flags.Bool("copy", false, "copy the prompt to the clipboard instead of stdout")
		noInput    = flags.Bool("no-input", false, "never prompt interactively; unsatisfied inputs error")
		jsonOut    = flags.Bool("json", false, "emit a stable JSON object (text, inputs, chars)")
		appID      = flags.String("app", "", "application id for app-id inputs (v1.4 tracker)")
	)
	// Register one flag per template input that is fed by a CLI flag.
	flagVals := map[string]*string{}
	for _, key := range sortedKeys(t.Inputs) {
		spec := t.Inputs[key]
		if spec.Source != prompt.SourceFlag && spec.Source != prompt.SourceJDFile {
			continue
		}
		fn := flagName(key, spec)
		if reservedRunFlags[fn] {
			return fmt.Errorf("template %q: input %q uses reserved flag --%s; give it a different flag name", t.Name, key, fn)
		}
		if _, exists := flagVals[fn]; exists {
			continue
		}
		flagVals[fn] = flags.String(fn, "", fmt.Sprintf("value for {{%s}}", key))
	}
	if helped, err := parseFlags(flags, rest); helped || err != nil {
		return err
	}
	if *output != "" && *copyOut {
		return usageErr(errors.New("--output and --copy are mutually exclusive"))
	}

	rc := resolveCtx{
		deps:     deps,
		appdir:   *appDirPath,
		profile:  *profile,
		appID:    *appID,
		noInput:  *noInput,
		flagVals: flagVals,
		stdin:    os.Stdin,
		reader:   bufio.NewReader(os.Stdin),
	}
	in := prompt.PromptInput{}
	for _, key := range sortedKeys(t.Inputs) {
		val, err := rc.resolve(ctx, key, t.Inputs[key])
		if err != nil {
			return err
		}
		in[key] = val
	}

	text, err := prompt.Render(t, in)
	if err != nil {
		return err
	}
	return emitPrompt(ctx, t, text, *output, *copyOut, *jsonOut)
}

// inputSources maps each input key to its source, for the run --json object.
func inputSources(t prompt.PromptTemplate) map[string]string {
	m := make(map[string]string, len(t.Inputs))
	for k, s := range t.Inputs {
		m[k] = s.Source
	}
	return m
}

// resolveCtx carries everything the per-input resolver needs.
type resolveCtx struct {
	deps     Deps
	appdir   string
	profile  string
	appID    string
	noInput  bool
	flagVals map[string]*string
	stdin    io.Reader     // raw stdin, used only to detect a terminal
	reader   *bufio.Reader // buffered view over stdin, shared across all inputs
}

func (rc resolveCtx) resolve(ctx context.Context, key string, spec prompt.InputSpec) (string, error) {
	switch spec.Source {
	case prompt.SourceFlag:
		val := rc.flagValue(key, spec)
		if val == "" {
			if fb, ok, err := rc.appFallback(ctx, key, spec); err != nil {
				return "", err
			} else if ok {
				return fb, nil
			}
		}
		return rc.orDefault(key, spec, val)
	case prompt.SourceJDFile:
		path := rc.flagValue(key, spec)
		if path == "" {
			if fb, ok, err := rc.appFallback(ctx, key, spec); err != nil {
				return "", err
			} else if ok {
				return fb, nil
			}
			return rc.requireOrDefault(key, spec, "")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("input %q: read %s: %w", key, path, err)
		}
		return string(raw), nil
	case prompt.SourceDataDump:
		return rc.readDataDump(ctx, key, spec)
	case prompt.SourceAppID:
		return rc.readAppField(ctx, key, spec)
	case prompt.SourceStdin:
		// Nothing is piped and we mustn't block: honor --no-input instead of
		// waiting on a terminal for EOF.
		if rc.noInput && isTerminal(rc.stdin) {
			return rc.requireOrDefault(key, spec, "")
		}
		// A non-TTY stdin can still block forever: a pipe held open by a
		// supervisor, CI runner, or agent harness never reaches EOF. --no-input
		// promises never to block, so wait only for the first byte. A real
		// producer delivers it immediately; an idle pipe never does. Once data
		// is there, read to EOF unbounded so a slow-but-real stream is safe.
		if rc.noInput && !rc.stdinHasData() {
			return rc.requireOrDefault(key, spec, "")
		}
		raw, err := io.ReadAll(rc.reader)
		if err != nil {
			return "", fmt.Errorf("input %q: read stdin: %w", key, err)
		}
		return rc.requireOrDefault(key, spec, strings.TrimRight(string(raw), "\n"))
	case prompt.SourcePrompt:
		return rc.readInteractive(key, spec)
	default:
		return "", fmt.Errorf("input %q: unsupported source %q", key, spec.Source)
	}
}

// resolveFromValues fills a template's inputs for the TUI: user-entered sources
// (flag/prompt/stdin, and jd-file as a typed path) come from the values map,
// while data-dump and app-id reuse the same resolvers the CLI uses. It never
// reads os.Stdin, so it is safe to call from inside the alt-screen TUI.
func (rc resolveCtx) resolveFromValues(ctx context.Context, t prompt.PromptTemplate, values map[string]string) (prompt.PromptInput, error) {
	in := prompt.PromptInput{}
	for key, spec := range t.Inputs {
		var (
			val string
			err error
		)
		switch spec.Source {
		case prompt.SourceFlag:
			v := strings.TrimSpace(values[key])
			if v == "" {
				if fb, ok, ferr := rc.appFallback(ctx, key, spec); ferr != nil {
					return nil, ferr
				} else if ok {
					in[key] = fb
					continue
				}
			}
			val, err = rc.requireOrDefault(key, spec, v)
		case prompt.SourcePrompt, prompt.SourceStdin:
			val, err = rc.requireOrDefault(key, spec, strings.TrimSpace(values[key]))
		case prompt.SourceJDFile:
			path := strings.TrimSpace(values[key])
			if path == "" {
				if fb, ok, ferr := rc.appFallback(ctx, key, spec); ferr != nil {
					return nil, ferr
				} else if ok {
					in[key] = fb
					continue
				}
				val, err = rc.requireOrDefault(key, spec, "")
				break
			}
			raw, rerr := os.ReadFile(path)
			if rerr != nil {
				return nil, fmt.Errorf("input %q: read %s: %w", key, path, rerr)
			}
			val = string(raw)
		case prompt.SourceDataDump:
			val, err = rc.readDataDump(ctx, key, spec)
		case prompt.SourceAppID:
			val, err = rc.readAppField(ctx, key, spec)
		default:
			err = fmt.Errorf("input %q: unsupported source %q", key, spec.Source)
		}
		if err != nil {
			return nil, err
		}
		in[key] = val
	}
	return in, nil
}

func (rc resolveCtx) flagValue(key string, spec prompt.InputSpec) string {
	if p, ok := rc.flagVals[flagName(key, spec)]; ok {
		return *p
	}
	return ""
}

func (rc resolveCtx) readDataDump(ctx context.Context, key string, spec prompt.InputSpec) (string, error) {
	cfgSource, res, err := layeredConfigSource(rc.appdir)
	if err != nil {
		return "", err
	}
	outputDir := "output"
	cfg, cerr := cfgSource.Load(ctx)
	if cerr == nil && cfg.Paths.OutputDir != "" {
		outputDir = cfg.Paths.OutputDir
	}
	path := filepath.Join(res.Dir, outputDir, rc.profile+".md")
	raw, err := os.ReadFile(path)
	if err != nil {
		if spec.Required {
			return "", fmt.Errorf("input %q: no data-dump at %s - enable [render] emit_markdown and render profile %q first", key, path, rc.profile)
		}
		return spec.Default, nil
	}
	return string(raw), nil
}

// readAppField resolves an app-id input: it loads the application named by --app
// (through track.Get, so lazy auto-ghost still applies) and returns the requested
// stored field. field="jd" reads the contents of the file at the app's jd_path.
func (rc resolveCtx) readAppField(ctx context.Context, key string, spec prompt.InputSpec) (string, error) {
	if rc.appID == "" {
		// No --app supplied: treat like any other unset input.
		return rc.requireOrDefault(key, spec, "")
	}
	tr, err := tracker(ctx, rc.deps, rc.appdir)
	if err != nil {
		return "", err
	}
	app, err := tr.Get(ctx, rc.appID)
	if err != nil {
		return "", fmt.Errorf("input %q: %w", key, err)
	}
	switch spec.Field {
	case "company":
		return rc.requireOrDefault(key, spec, app.Company)
	case "role":
		return rc.requireOrDefault(key, spec, app.Role)
	case "status":
		return rc.requireOrDefault(key, spec, string(app.Status))
	case "source":
		return rc.requireOrDefault(key, spec, app.Source)
	case "notes":
		return rc.requireOrDefault(key, spec, app.Notes)
	case "jd":
		if app.JDPath == "" {
			return rc.requireOrDefault(key, spec, "")
		}
		raw, err := os.ReadFile(app.JDPath)
		if err != nil {
			return "", fmt.Errorf("input %q: read jd_path %s: %w", key, app.JDPath, err)
		}
		return string(raw), nil
	default:
		return "", fmt.Errorf("input %q: unknown app-id field %q", key, spec.Field)
	}
}

// appFallback fills an EMPTY flag/jd-file input from the application named by
// --app, so `prompt run <name> --app <id>` works with every template exactly as
// the README promises: company/role flag inputs take the app's fields, and an
// empty jd-file input reads the file at the app's jd_path. Explicit values
// always win; without --app nothing changes.
func (rc resolveCtx) appFallback(ctx context.Context, key string, spec prompt.InputSpec) (string, bool, error) {
	if rc.appID == "" {
		return "", false, nil
	}
	// Eligibility first: only jd-file inputs and company/role flags can take an
	// app fallback. Inputs that never touch the application must not fail (or
	// pay a load) because of a stray or bad --app.
	isJD := spec.Source == prompt.SourceJDFile
	fn := flagName(key, spec)
	isField := spec.Source == prompt.SourceFlag && (fn == "company" || fn == "role")
	if !isJD && !isField {
		return "", false, nil
	}
	tr, err := tracker(ctx, rc.deps, rc.appdir)
	if err != nil {
		return "", false, err
	}
	app, err := tr.Get(ctx, rc.appID)
	if err != nil {
		return "", false, fmt.Errorf("--app: %w", err)
	}
	if isJD {
		if app.JDPath == "" {
			return "", false, nil
		}
		raw, err := os.ReadFile(app.JDPath)
		if err != nil {
			return "", false, fmt.Errorf("input %q: read jd_path %s: %w", key, app.JDPath, err)
		}
		return string(raw), true, nil
	}
	if fn == "company" {
		return app.Company, app.Company != "", nil
	}
	return app.Role, app.Role != "", nil
}

func (rc resolveCtx) readInteractive(key string, spec prompt.InputSpec) (string, error) {
	if rc.noInput {
		if spec.Default != "" {
			return spec.Default, nil
		}
		if spec.Required {
			return "", fmt.Errorf("input %q needs interactive entry but --no-input is set", key)
		}
		return "", nil
	}
	fmt.Fprintf(os.Stderr, "%s: ", key)
	line, err := rc.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("input %q: read: %w", key, err)
	}
	return rc.requireOrDefault(key, spec, strings.TrimRight(line, "\n"))
}

// orDefault applies the spec default to an empty value and enforces required.
func (rc resolveCtx) orDefault(key string, spec prompt.InputSpec, val string) (string, error) {
	return rc.requireOrDefault(key, spec, val)
}

func (rc resolveCtx) requireOrDefault(key string, spec prompt.InputSpec, val string) (string, error) {
	if val != "" {
		return val, nil
	}
	if spec.Default != "" {
		return spec.Default, nil
	}
	if spec.Required {
		return "", fmt.Errorf("required input %q is unset (%s)", key, sourceHint(spec))
	}
	return "", nil
}

func emitPrompt(ctx context.Context, t prompt.PromptTemplate, text, output string, copyOut, jsonOut bool) error {
	// Side effect (clipboard/file) happens regardless of output format.
	obj := runJSON{Prompt: t.Name, Text: text, Inputs: inputSources(t), Chars: len(text)}
	switch {
	case copyOut:
		if err := clipboard.Copy(ctx, text); err != nil {
			return err
		}
		obj.Copied = true
	case output != "":
		if err := os.WriteFile(output, []byte(text), 0o600); err != nil {
			return fmt.Errorf("write %s: %w", output, err)
		}
		obj.Output = output
	}

	if jsonOut {
		// In --json mode the confirmation lives in the object; stdout stays parseable.
		return emitJSON(obj)
	}
	switch {
	case copyOut:
		fmt.Fprintf(os.Stderr, "copied to clipboard (%d chars)\n", len(text))
	case output != "":
		fmt.Fprintf(os.Stderr, "wrote %s (%d chars)\n", output, len(text))
	default:
		fmt.Print(text)
	}
	return nil
}

// flagName returns the CLI flag that feeds an input: its explicit Flag, else the
// input key.
func flagName(key string, spec prompt.InputSpec) string {
	if spec.Flag != "" {
		return spec.Flag
	}
	return key
}

func sourceHint(spec prompt.InputSpec) string {
	if spec.Flag != "" {
		return "--" + spec.Flag
	}
	return "source " + spec.Source
}

// scanFlag does a permissive left-to-right scan for --name / -name / =value,
// used to peek at --path before the real FlagSet (which needs the template's
// dynamic flags) is built.
func scanFlag(args []string, name string) string {
	for i := 0; i < len(args); i++ {
		a := args[i]
		for _, pfx := range []string{"--" + name, "-" + name} {
			if a == pfx {
				if i+1 < len(args) {
					return args[i+1]
				}
				return ""
			}
			if strings.HasPrefix(a, pfx+"=") {
				return strings.TrimPrefix(a, pfx+"=")
			}
		}
	}
	return ""
}

// stdinFirstByteWait bounds how long --no-input waits for stdin to produce its
// first byte before concluding nothing is piped. Only the first byte is
// bounded, so a slow producer is never truncated.
// Overridable so tests need not wait the real bound.
var stdinFirstByteWait = 2 * time.Second

// stdinHasData reports whether stdin yields at least one byte within
// stdinFirstByteWait. Peek does not consume, so a later ReadAll still sees the
// full stream.
//
// The peek runs in a goroutine because a blocked read on a pipe cannot be
// cancelled: on timeout that goroutine stays parked until the process exits,
// which is fine for a short-lived CLI and is the price of not hanging forever.
func (rc resolveCtx) stdinHasData() bool {
	type peekResult struct{ err error }
	done := make(chan peekResult, 1) // buffered: the goroutine never blocks on send
	go func() {
		_, err := rc.reader.Peek(1)
		done <- peekResult{err}
	}()
	select {
	case r := <-done:
		// EOF (closed pipe, /dev/null) means genuinely no input.
		return r.err == nil
	case <-time.After(stdinFirstByteWait):
		return false
	}
}

// isTerminal reports whether r is a real interactive terminal. A non-*os.File
// reader (test buffer, pipe) - and a char device that is not a TTY, like
// /dev/null - is never a terminal.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
