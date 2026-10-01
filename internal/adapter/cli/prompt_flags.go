package cli

import (
	"strings"

	"github.com/crispuscrew/resumegen/internal/usecase/prompt"
)

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

// Peek at --path before loading the template's dynamic flags.
func scanFlag(args []string, name string) string {
	for index := 0; index < len(args); index++ {
		argument := args[index]
		for _, prefix := range []string{"--" + name, "-" + name} {
			if argument == prefix {
				if index+1 < len(args) {
					return args[index+1]
				}
				return ""
			}
			if strings.HasPrefix(argument, prefix+"=") {
				return strings.TrimPrefix(argument, prefix+"=")
			}
		}
	}
	return ""
}
