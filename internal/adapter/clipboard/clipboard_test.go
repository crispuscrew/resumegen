package clipboard

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestCopy_NoToolAvailable(t *testing.T) {
	orig := lookPath
	defer func() { lookPath = orig }()
	lookPath = func(string) (string, error) { return "", errors.New("not found") }

	err := Copy(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error when no clipboard tool is present")
	}
	if !strings.Contains(err.Error(), "wl-clipboard") || !strings.Contains(err.Error(), "xclip") {
		t.Errorf("error should name both helpers and be actionable, got: %v", err)
	}
}

func TestCopy_UsesFirstAvailable(t *testing.T) {
	orig := lookPath
	defer func() { lookPath = orig }()
	// wl-copy "missing", xclip resolves to /bin/true (accepts stdin, exits 0).
	truePath, err := exec.LookPath("true")
	if err != nil {
		t.Skip("no `true` binary to stand in for a clipboard tool")
	}
	lookPath = func(bin string) (string, error) {
		if bin == "xclip" {
			return truePath, nil
		}
		return "", errors.New("not found")
	}

	if err := Copy(context.Background(), "hello"); err != nil {
		t.Errorf("expected success via the second helper, got %v", err)
	}
}
