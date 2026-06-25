package container

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fakeRunner records calls and returns canned results.
type fakeRunner struct {
	calls []call
	// runErrs is consumed in order on each Run call.
	runErrs []error
}

type call struct {
	bin  string
	args []string
}

func (f *fakeRunner) Run(_ context.Context, bin string, args []string, _ io.Writer, _ io.Writer) error {
	f.calls = append(f.calls, call{bin, append([]string(nil), args...)})
	if len(f.runErrs) == 0 {
		return nil
	}
	err := f.runErrs[0]
	f.runErrs = f.runErrs[1:]
	return err
}

func (f *fakeRunner) Output(_ context.Context, bin string, args []string) ([]byte, error) {
	f.calls = append(f.calls, call{bin, append([]string(nil), args...)})
	return nil, nil
}

// exitErr satisfies the interface{ ExitCode() int } shape that isExitError checks.
type exitErr struct{ code int }

func (e exitErr) Error() string  { return "exit" }
func (e exitErr) ExitCode() int  { return e.code }

func TestImageTag(t *testing.T) {
	cases := []struct{ in, want string }{
		{"1.1.0", "localhost/resumegen-render:1.1.0"},
		{"", "localhost/resumegen-render:dev"},
	}
	for _, c := range cases {
		if got := ImageTag(c.in); got != c.want {
			t.Errorf("ImageTag(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestImageExists_True(t *testing.T) {
	r := &fakeRunner{runErrs: []error{nil}}
	ok, err := ImageExists(context.Background(), r, Engine{Name: "podman", Bin: "podman"}, "img")
	if err != nil || !ok {
		t.Fatalf("got (%v,%v), want (true,nil)", ok, err)
	}
	if !reflect.DeepEqual(r.calls[0].args, []string{"image", "exists", "img"}) {
		t.Errorf("unexpected args: %v", r.calls[0].args)
	}
}

func TestImageExists_FalseOnExitError(t *testing.T) {
	r := &fakeRunner{runErrs: []error{exitErr{1}}}
	ok, err := ImageExists(context.Background(), r, Engine{Name: "podman", Bin: "podman"}, "img")
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if ok {
		t.Fatal("expected ok=false")
	}
}

func TestImageExists_PropagatesNonExitErr(t *testing.T) {
	boom := errors.New("exec boom")
	r := &fakeRunner{runErrs: []error{boom}}
	_, err := ImageExists(context.Background(), r, Engine{Name: "podman", Bin: "podman"}, "img")
	if err == nil || !strings.Contains(err.Error(), "exec boom") {
		t.Fatalf("err = %v, want one wrapping %q", err, "exec boom")
	}
}

func TestBuildImage_WritesContainerfileAndRunsEngine(t *testing.T) {
	var cfPath string
	r := &fakeRunner{}
	// inspect the tmp path via the recorded call (-f arg)
	err := BuildImage(context.Background(), r, Engine{Name: "podman", Bin: "podman"},
		"localhost/x:1", []byte("FROM alpine\n"), &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("BuildImage err: %v", err)
	}
	if len(r.calls) != 1 {
		t.Fatalf("call count = %d, want 1", len(r.calls))
	}
	args := r.calls[0].args
	if args[0] != "build" || args[1] != "-t" || args[2] != "localhost/x:1" || args[3] != "-f" {
		t.Fatalf("argv prefix = %v", args[:4])
	}
	cfPath = args[4]
	// file should have already been removed by defer — only the tmp dir leak would surface.
	if filepath.Base(cfPath) != "Containerfile" {
		t.Errorf("expected -f to point at Containerfile, got %q", cfPath)
	}
	// confirm cleanup removed the dir
	if _, err := os.Stat(filepath.Dir(cfPath)); !os.IsNotExist(err) {
		t.Errorf("tmpdir leaked: %v", err)
	}
}

func TestEnsureImage_SkipsBuildWhenPresent(t *testing.T) {
	r := &fakeRunner{runErrs: []error{nil}} // first call (exists) succeeds
	err := EnsureImage(context.Background(), r, Engine{Name: "podman", Bin: "podman"},
		"img", []byte("FROM x"), io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 1 {
		t.Errorf("expected 1 call (exists check), got %d", len(r.calls))
	}
}

func TestEnsureImage_BuildsWhenMissing(t *testing.T) {
	r := &fakeRunner{runErrs: []error{exitErr{1}, nil}} // exists=miss, build=ok
	err := EnsureImage(context.Background(), r, Engine{Name: "podman", Bin: "podman"},
		"img", []byte("FROM x"), io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 2 {
		t.Errorf("expected 2 calls (exists + build), got %d", len(r.calls))
	}
	if r.calls[1].args[0] != "build" {
		t.Errorf("second call should be build, got %v", r.calls[1].args)
	}
}
