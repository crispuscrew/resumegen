package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

// Template inputs may not collide with built-in flags.
var reservedRunFlags = map[string]bool{
	"path": true, "profile": true, "output": true,
	"copy": true, "no-input": true, "json": true, "app": true,
}

func promptRun(ctx context.Context, deps Deps, args []string) error {
	if helped, err := positionalArgs(args, 1, "resumegen prompt run <name> [flags]"); helped || err != nil {
		return err
	}
	name, rest := args[0], args[1:]
	// The template must be loaded before registering its dynamic flags.
	repo, err := promptRepo(deps, scanFlag(rest, "path"))
	if err != nil {
		return err
	}
	template, err := repo.Load(ctx, name)
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
	flagVals := map[string]*string{}
	for _, key := range sortedKeys(template.Inputs) {
		spec := template.Inputs[key]
		if spec.Source != prompt.SourceFlag && spec.Source != prompt.SourceJDFile && spec.Source != prompt.SourceFile {
			continue
		}
		name := flagName(key, spec)
		if reservedRunFlags[name] {
			return fmt.Errorf("template %q: input %q uses reserved flag --%s; give it a different flag name", template.Name, key, name)
		}
		if _, exists := flagVals[name]; exists {
			continue
		}
		flagVals[name] = flags.String(name, "", fmt.Sprintf("value for {{%s}}", key))
	}
	if helped, err := parseFlags(flags, rest); helped || err != nil {
		return err
	}
	if *output != "" && *copyOut {
		return usageErr(errors.New("--output and --copy are mutually exclusive"))
	}
	resolver := resolveCtx{
		deps: deps, appdir: *appDirPath, profile: *profile, appID: *appID,
		noInput: *noInput, flagVals: flagVals, stdin: os.Stdin,
		reader: bufio.NewReader(os.Stdin),
	}
	inputs := prompt.PromptInput{}
	for _, key := range sortedKeys(template.Inputs) {
		value, err := resolver.resolve(ctx, key, template.Inputs[key])
		if err != nil {
			return err
		}
		inputs[key] = value
	}
	text, err := prompt.Render(template, inputs)
	if err != nil {
		return err
	}
	return emitPrompt(ctx, template, text, *output, *copyOut, *jsonOut)
}
