// Package container implements usecase.Renderer by running typst inside a
// rootless container. The supported engines are podman (preferred) and docker
// (fallback). All security flags are fixed: --read-only, --network=none,
// --cap-drop=ALL, --security-opt=no-new-privileges, non-root user.
package container

import (
	"fmt"
	"os/exec"
)

// Engine names a detected container runtime.
type Engine struct {
	Name string // "podman" | "docker"
	Bin  string // absolute path to the binary
}

// Detector probes for an engine on PATH. LookPath is injectable for tests.
type Detector struct {
	LookPath func(string) (string, error)
}

// Detect returns the first available engine in priority order
// (podman, then docker) and a boolean indicating success.
func (d Detector) Detect() (Engine, bool) {
	lookPath := d.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	for _, name := range []string{"podman", "docker"} {
		if bin, err := lookPath(name); err == nil {
			return Engine{Name: name, Bin: bin}, true
		}
	}
	return Engine{}, false
}

// RunSpec describes a single `<engine> run` invocation.
type RunSpec struct {
	Image     string   // full image reference, e.g. "localhost/resumegen-render:1.1.0"
	AppdirRO  string   // host path mounted read-only at /work
	OutputRW  string   // host path mounted read-write at /work/output
	UID, GID  int      // numeric ids used for --user (host uid/gid)
	TypstArgs []string // appended after the image; e.g. ["compile", "/work/templates/resume.typ", "/work/output/x.pdf"]
}

// RunArgs renders the engine-specific argv for `<engine> run ...`. It does not
// include the engine binary itself (caller calls exec.CommandContext(eng.Bin, args...)).
func (e Engine) RunArgs(spec RunSpec) []string {
	args := []string{
		"run", "--rm",
		"--read-only",
		"--network=none",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--tmpfs=/tmp:rw,size=64m",
		"--user", fmt.Sprintf("%d:%d", spec.UID, spec.GID),
	}
	if e.Name == "podman" {
		args = append(args, "--userns=keep-id")
	}
	args = append(args,
		"-v", spec.AppdirRO+":/work:ro,Z",
		"-v", spec.OutputRW+":/work/output:rw,Z",
		spec.Image,
	)
	return append(args, spec.TypstArgs...)
}

// BuildImageArgs renders the argv for `<engine> build -t <tag> -f <file> <ctx>`.
func (e Engine) BuildImageArgs(containerfile, contextDir, tag string) []string {
	return []string{"build", "-t", tag, "-f", containerfile, contextDir}
}

// ImageExistsArgs renders the argv that exits 0 iff the named image exists
// locally. For podman, `image exists` is purpose-built; for docker we use
// `image inspect`, which prints metadata and is silenced by the caller.
func (e Engine) ImageExistsArgs(tag string) []string {
	if e.Name == "podman" {
		return []string{"image", "exists", tag}
	}
	return []string{"image", "inspect", tag}
}
