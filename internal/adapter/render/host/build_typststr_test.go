package host

import "testing"

func TestTypstStr(t *testing.T) {
	cases := map[string]string{
		`plain`:        `"plain"`,
		`say "hi"`:     `"say \"hi\""`,
		`back\slash`:   `"back\\slash"`,
		"tab\there":    `"tab\there"`,
		"line\nbreak":  `"line\nbreak"`,
		"bell\x07ring": `"bell\u{7}ring"`, // Go %q would emit \x07 - invalid Typst
		"Привет":       `"Привет"`,        // printable unicode passes raw
	}
	for in, want := range cases {
		if got := typstStr(in); got != want {
			t.Errorf("typstStr(%q) = %s, want %s", in, got, want)
		}
	}
}
