package container

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestDetect_PodmanFirst(t *testing.T) {
	d := Detector{LookPath: func(name string) (string, error) {
		switch name {
		case "podman":
			return "/usr/bin/podman", nil
		case "docker":
			return "/usr/bin/docker", nil
		}
		return "", errors.New("not found")
	}}
	eng, ok := d.Detect()
	if !ok {
		t.Fatal("expected detection to succeed")
	}
	if eng.Name != "podman" || eng.Bin != "/usr/bin/podman" {
		t.Errorf("got %+v, want podman /usr/bin/podman", eng)
	}
}

func TestDetect_DockerFallback(t *testing.T) {
	d := Detector{LookPath: func(name string) (string, error) {
		if name == "docker" {
			return "/usr/bin/docker", nil
		}
		return "", errors.New("not found")
	}}
	eng, ok := d.Detect()
	if !ok {
		t.Fatal("expected detection to succeed")
	}
	if eng.Name != "docker" {
		t.Errorf("got %+v, want docker", eng)
	}
}

func TestDetect_None(t *testing.T) {
	d := Detector{LookPath: func(string) (string, error) { return "", errors.New("nope") }}
	_, ok := d.Detect()
	if ok {
		t.Fatal("expected no engine")
	}
}

func TestRunArgs_Podman(t *testing.T) {
	e := Engine{Name: "podman", Bin: "/usr/bin/podman"}
	args := e.RunArgs(RunSpec{
		Image:     "localhost/resumegen-render:1.1.0",
		AppdirRO:  "/home/u/.config/resumegen",
		OutputRW:  "/home/u/.config/resumegen/output",
		UID:       1000,
		GID:       1000,
		TypstArgs: []string{"compile", "/work/templates/resume.typ", "/work/output/x.pdf"},
	})
	want := []string{
		"run", "--rm",
		"--read-only",
		"--network=none",
		"--cap-drop=ALL",
		"--security-opt=no-new-privileges",
		"--tmpfs=/tmp:rw,size=64m",
		"--user", "1000:1000",
		"--userns=keep-id",
		"-v", "/home/u/.config/resumegen:/work:ro,Z",
		"-v", "/home/u/.config/resumegen/output:/work/output:rw,Z",
		"localhost/resumegen-render:1.1.0",
		"compile", "/work/templates/resume.typ", "/work/output/x.pdf",
	}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("args mismatch:\n got %s\nwant %s",
			strings.Join(args, " "),
			strings.Join(want, " "))
	}
}

func TestRunArgs_Docker_NoUserns(t *testing.T) {
	e := Engine{Name: "docker", Bin: "/usr/bin/docker"}
	args := e.RunArgs(RunSpec{
		Image: "img", AppdirRO: "/a", OutputRW: "/a/output",
		UID: 1000, GID: 1000, TypstArgs: []string{"compile"},
	})
	for _, a := range args {
		if a == "--userns=keep-id" {
			t.Fatal("docker run must not include --userns=keep-id")
		}
	}
}

func TestImageExistsArgs(t *testing.T) {
	if got := (Engine{Name: "podman"}).ImageExistsArgs("img"); !reflect.DeepEqual(got, []string{"image", "exists", "img"}) {
		t.Errorf("podman: got %v", got)
	}
	if got := (Engine{Name: "docker"}).ImageExistsArgs("img"); !reflect.DeepEqual(got, []string{"image", "inspect", "img"}) {
		t.Errorf("docker: got %v", got)
	}
}

func TestBuildImageArgs(t *testing.T) {
	args := Engine{Name: "podman"}.BuildImageArgs("/tmp/Containerfile", "/tmp/ctx", "img:tag")
	want := []string{"build", "-t", "img:tag", "-f", "/tmp/Containerfile", "/tmp/ctx"}
	if !reflect.DeepEqual(args, want) {
		t.Errorf("got %v, want %v", args, want)
	}
}
