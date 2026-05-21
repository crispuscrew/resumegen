package sanitize

import (
	"strings"
	"testing"
)

// Happy-path table: known-good markup forms from the default appdir
// produce stable, expected Typst output.
func TestSanitize_HappyPath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain text", "Hello world", "Hello world"},
		{"bold", "*REST API*", "*REST API*"},
		{"italic", "_emphasis_", "_emphasis_"},
		{"code", "`code`", "`code`"},
		{"bold around plain", "use *gRPC* daily", "use *gRPC* daily"},
		{"code inside bold", "*`<100 ms`*", "*`<100 ms`*"},
		{"new link syntax", "[GitHub](https://github.com/x)", `#link("https://github.com/x")[GitHub]`},
		{"legacy link syntax", `#link("https://x")[x]`, `#link("https://x")[x]`},
		{"mailto link", "[mail](mailto:a@b.com)", `#link("mailto:a@b.com")[mail]`},
		{"colon and hyphen pass through", "a:b - c", "a:b - c"},
		{"plus signs unchanged", "C++ rocks", "C++ rocks"},
		{"unicode passthrough", "↔ Привет", "↔ Привет"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Sanitize(tc.in, Strict)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q want %q", got, tc.want)
			}
		})
	}
}

// Escaping table: every Typst metachar listed in DESIGN §4.2 step 3 (single
// char) gets a leading backslash when emitted as plain text.
func TestSanitize_EscapesTypstMetachars(t *testing.T) {
	cases := []struct{ ch, want string }{
		{`\`, `\\`},
		{`"`, `\"`},
		{`#`, `\#`},
		{`<`, `\<`},
		{`>`, `\>`},
		{`@`, `\@`},
		{`=`, `\=`},
		{`[`, `\[`},
		{`]`, `\]`},
		{`(`, `\(`},
		{`)`, `\)`},
		{`~`, `\~`},
	}
	for _, tc := range cases {
		// Wrap each char with neutral text so the parser sees it as plain.
		in := "x" + tc.ch + "y"
		want := "x" + tc.want + "y"
		got, err := Sanitize(in, Strict)
		if err != nil {
			t.Fatalf("char %q: %v", tc.ch, err)
		}
		if got != want {
			t.Errorf("char %q: got %q want %q", tc.ch, got, want)
		}
	}
}

// Injection corpus: every form an attacker might try to break out of the
// content block and into Typst code mode. All must either fail in strict
// mode or be neutralized via escaping. "Neutralized" means every Typst
// metachar in the output has a leading backslash.
func TestSanitize_InjectionAttempts(t *testing.T) {
	mustBeEscaped := []byte{'#', '<', '>', '(', ')', '[', ']', '@', '=', '*', '_', '`', '~'}
	assertNoBareMetachars := func(t *testing.T, in, out string) {
		t.Helper()
		for i := 0; i < len(out); i++ {
			b := out[i]
			bare := false
			for _, m := range mustBeEscaped {
				if b == m {
					bare = true
					break
				}
			}
			if !bare {
				continue
			}
			// Inside Bold/Italic/Code/Link spans, the delimiter itself is
			// supposed to appear bare — but the corpus below only contains
			// plain-text injection attempts (no legitimate markup), so any
			// metachar in the output must be backslash-prefixed.
			if i == 0 || out[i-1] != '\\' {
				t.Errorf("input %q produced bare metachar %q at offset %d in output %q", in, b, i, out)
				return
			}
		}
	}
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{name: "raw # function call gets escaped", in: `#read("/etc/passwd")`},
		{name: "raw label syntax neutralized", in: "go to <secret-label>"},
		{name: "let assignment attempt neutralized", in: "#let x = 1; #x"},
		{name: "raw bracket sequence as plain text", in: "see [1] for details"},
		{
			name:    "javascript: in legacy link rejected",
			in:      `#link("javascript:alert(1)")[click]`,
			wantErr: true,
		},
		{
			name: "data: in legacy link rejected",
			in:   `#link("data:text/html,<script>")[click]`,
			wantErr: true,
		},
		{
			name: "file:// in legacy link rejected",
			in:   `#link("file:///etc/passwd")[oops]`,
			wantErr: true,
		},
		{
			name: "javascript: in new link rejected",
			in:   `[click](javascript:alert(1))`,
			wantErr: true,
		},
		{
			name: "ssh:// in new link rejected",
			in:   `[clone](ssh://git@example.com/x.git)`,
			wantErr: true,
		},
		{
			name: "path-traversal-y URL still has to pass scheme check",
			in:   `[bad](../../etc/passwd)`,
			wantErr: true,
		},
		{
			name:    "NUL byte rejected",
			in:      "good\x00evil",
			wantErr: true,
		},
		{
			name:    "invalid UTF-8 rejected",
			in:      "good\xc3\x28text",
			wantErr: true,
		},
		{
			name: "unterminated bold rejected",
			in:   "*foo",
			wantErr: true,
		},
		{
			name: "unterminated italic rejected",
			in:   "_bar",
			wantErr: true,
		},
		{
			name: "unterminated code rejected",
			in:   "`baz",
			wantErr: true,
		},
		{
			name: "control character (BEL) in URL rejected",
			in:   "[x](http://exa\x07mple.com)",
			wantErr: true,
		},
		{
			name: "newline in URL rejected",
			in:   "[x](http://example.com/\n)",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Sanitize(tc.in, Strict)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertNoBareMetachars(t, tc.in, got)
		})
	}
}

// Oversized input: the sanitizer must not panic or time out on large blobs.
// Length-limit gating is slice-5 (strict_input); here we just confirm the
// parser is linear and finishes.
func TestSanitize_LargeInputDoesNotPanic(t *testing.T) {
	in := strings.Repeat("a", 1<<16)
	if _, err := Sanitize(in, Strict); err != nil {
		t.Fatalf("unexpected error on 64KiB plain input: %v", err)
	}
}

// Permissive mode falls back to fully-escaped literal on any parse failure
// instead of propagating the error.
func TestSanitize_PermissiveRecoversFromBadInput(t *testing.T) {
	cases := []string{
		"*unterminated bold",
		`#link("javascript:bad")[hi]`,
		"`code never ends",
	}
	for _, in := range cases {
		got, err := Sanitize(in, Permissive)
		if err != nil {
			t.Errorf("permissive mode returned error for %q: %v", in, err)
			continue
		}
		// Output must be safe (no raw # or unbalanced markup chars that
		// would land in Typst as syntax).
		if strings.Contains(got, "#link(\"javascript") {
			t.Errorf("permissive mode leaked unsafe URL in %q -> %q", in, got)
		}
	}
}

// Lone '[' that doesn't open a link is treated as literal — not an error.
// Default-appdir data uses brackets in prose like "[1] reference".
func TestSanitize_BracketWithoutLinkIsLiteral(t *testing.T) {
	got, err := Sanitize("see [1] for details", Strict)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `\[1\]`) {
		t.Errorf("got %q, expected \\[1\\] escaping", got)
	}
}

// Empty input renders to empty output without error.
func TestSanitize_EmptyInput(t *testing.T) {
	got, err := Sanitize("", Strict)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("got %q want empty", got)
	}
}

// URL allowlist is exhaustive — only http, https, mailto pass.
func TestValidateURL(t *testing.T) {
	good := []string{
		"http://example.com",
		"https://example.com/path?q=1",
		"mailto:a@b.com",
		"HTTPS://EXAMPLE.COM",
	}
	for _, u := range good {
		if err := validateURL(u); err != nil {
			t.Errorf("%q: expected nil, got %v", u, err)
		}
	}
	bad := []string{
		"", "javascript:alert(1)", "data:,x", "file:///etc/passwd",
		"ssh://git@host/repo", "ftp://example.com", "/relative",
		"http://example.com/\x00", "http://example.com/\n",
	}
	for _, u := range bad {
		if err := validateURL(u); err == nil {
			t.Errorf("%q: expected error, got nil", u)
		}
	}
}
