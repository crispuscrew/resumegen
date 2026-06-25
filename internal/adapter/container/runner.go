package container

import (
	"context"
	"io"
	"os/exec"
)

// Runner abstracts the execution of an external command. Tests inject a fake
// to record argv and return canned exit codes/outputs without spawning podman.
type Runner interface {
	Run(ctx context.Context, bin string, args []string, stdout, stderr io.Writer) error
	Output(ctx context.Context, bin string, args []string) (stdout []byte, err error)
}

// ExecRunner is the default Runner; it uses os/exec.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, bin string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func (ExecRunner) Output(ctx context.Context, bin string, args []string) ([]byte, error) {
	return exec.CommandContext(ctx, bin, args...).Output()
}
