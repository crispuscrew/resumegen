package container

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ImageTag derives the local-only image reference for a given resumegen
// version. The `localhost/` prefix prevents accidental registry pulls.
func ImageTag(version string) string {
	v := version
	if v == "" {
		v = "dev"
	}
	return "localhost/resumegen-render:" + v
}

// ImageExists returns true if the named image is present in the engine's
// local store.
func ImageExists(ctx context.Context, runner Runner, eng Engine, tag string) (bool, error) {
	args := eng.ImageExistsArgs(tag)
	err := runner.Run(ctx, eng.Bin, args, io.Discard, io.Discard)
	if err == nil {
		return true, nil
	}
	if isExitError(err) {
		return false, nil
	}
	return false, fmt.Errorf("%s %s: %w", eng.Name, joinArgs(args), err)
}

// BuildImage writes containerfile to a tmpdir as `Containerfile`, then runs
// `<engine> build -t <tag> -f <Containerfile> <ctx>`. Build stdout/stderr are
// forwarded to the supplied writers so the user can see progress.
func BuildImage(ctx context.Context, runner Runner, eng Engine, tag string, containerfile []byte, stdout, stderr io.Writer) error {
	dir, err := os.MkdirTemp("", "resumegen-build-")
	if err != nil {
		return fmt.Errorf("mkdir tmp: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	cfPath := filepath.Join(dir, "Containerfile")
	if err := os.WriteFile(cfPath, containerfile, 0o644); err != nil {
		return fmt.Errorf("write Containerfile: %w", err)
	}

	args := eng.BuildImageArgs(cfPath, dir, tag)
	if err := runner.Run(ctx, eng.Bin, args, stdout, stderr); err != nil {
		return fmt.Errorf("%s build: %w", eng.Name, err)
	}
	return nil
}

// EnsureImage builds the image only if it is not already present.
func EnsureImage(ctx context.Context, runner Runner, eng Engine, tag string, containerfile []byte, stdout, stderr io.Writer) error {
	ok, err := ImageExists(ctx, runner, eng, tag)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return BuildImage(ctx, runner, eng, tag, containerfile, stdout, stderr)
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}

// isExitError reports whether err is a non-zero exit from the spawned process
// (as opposed to an exec failure like missing binary). The exec package
// returns *exec.ExitError for the former.
func isExitError(err error) bool {
	type exitCoder interface{ ExitCode() int }
	if ec, ok := err.(exitCoder); ok {
		return ec.ExitCode() != 0
	}
	return false
}
