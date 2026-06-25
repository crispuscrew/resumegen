package main

import (
	"io/fs"
	"log"

	"github.com/crispuscrew/resumegen"
	"github.com/crispuscrew/resumegen/internal/adapter/cli"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	skeleton, err := fs.Sub(resumegen.Defaults, "defaultAppDir")
	if err != nil {
		log.Fatalf("embedded skeleton: %v", err)
	}
	cli.Run(cli.Deps{
		Version:           version,
		Skeleton:          skeleton,
		ContainerfileRend: resumegen.ContainerfileRender,
	})
}
