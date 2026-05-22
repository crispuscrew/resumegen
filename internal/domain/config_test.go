package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestParseContainerMode(t *testing.T) {
	cases := []struct {
		in   string
		want ContainerMode
	}{
		{"", ContainerOff},
		{"false", ContainerOff},
		{"true", ContainerOn},
		{"auto", ContainerAuto},
	}
	for _, c := range cases {
		got, err := ParseContainerMode(c.in)
		if err != nil {
			t.Errorf("ParseContainerMode(%q) error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseContainerMode(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseContainerMode_Invalid(t *testing.T) {
	_, err := ParseContainerMode("yes")
	if err == nil {
		t.Fatal("expected error for invalid value")
	}
	var ice *InvalidContainerModeError
	if !errors.As(err, &ice) {
		t.Fatalf("got %T, want *InvalidContainerModeError", err)
	}
	if ice.Value != "yes" {
		t.Errorf("Value = %q, want %q", ice.Value, "yes")
	}
	if !strings.Contains(err.Error(), `"yes"`) {
		t.Errorf("Error() should quote the bad value; got %q", err.Error())
	}
}
