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

func TestTracker_WithDefaults(t *testing.T) {
	// zero value fills both defaults
	got := (Tracker{}).WithDefaults()
	if got.GhostAfterDays != 30 || got.FollowupDefaultLagDays != 7 {
		t.Errorf("defaults = %+v, want {30, 7}", got)
	}
	// explicit values are preserved
	got = Tracker{GhostAfterDays: 14, FollowupDefaultLagDays: 3}.WithDefaults()
	if got.GhostAfterDays != 14 || got.FollowupDefaultLagDays != 3 {
		t.Errorf("explicit = %+v, want {14, 3}", got)
	}
	// a single zero field falls back independently
	got = Tracker{GhostAfterDays: 90}.WithDefaults()
	if got.GhostAfterDays != 90 || got.FollowupDefaultLagDays != 7 {
		t.Errorf("partial = %+v, want {90, 7}", got)
	}
}

func TestConfig_WithDefaults(t *testing.T) {
	// zero config gets the operational defaults
	got := (Config{}).WithDefaults()
	if got.Paths.TypstBin != "typst" || got.Paths.OutputDir != "output" {
		t.Errorf("paths defaults = %+v", got.Paths)
	}
	if got.Render.PageLimit != 1.0 || got.Render.PageHeightPt != 841.89 {
		t.Errorf("render defaults = %+v", got.Render)
	}
	// explicit values survive
	c := Config{}
	c.Paths.TypstBin = "/opt/typst"
	c.Render.PageLimit = 2.0
	got = c.WithDefaults()
	if got.Paths.TypstBin != "/opt/typst" || got.Render.PageLimit != 2.0 {
		t.Errorf("explicit values clobbered: %+v", got)
	}
	// toggles are NOT defaulted (a partial config must not flip behavior)
	if got.Render.StripMetadata || got.Render.EmitMarkdown || got.Score.SkillPriority != 0 {
		t.Errorf("behavioral toggles must stay zero: %+v", got)
	}
}

func TestTUI_ResolvedTheme(t *testing.T) {
	cases := map[string]string{
		"":        "default", // unset
		"default": "default", // explicit
		"solaris": "default", // unknown falls back
	}
	for in, want := range cases {
		if got := (TUI{Theme: in}).ResolvedTheme(); got != want {
			t.Errorf("ResolvedTheme(%q) = %q, want %q", in, got, want)
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
