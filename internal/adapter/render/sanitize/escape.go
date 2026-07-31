package sanitize

import "strings"

// escapeTypstContent escapes the Typst markup metachars listed in DESIGN
// section 4.2 step 3 so the result is safe to inline as literal text inside a
// content block `[...]`. Backslash is escaped first to keep the operation
// idempotent. Multi-char patterns (`--`, `---`) are NOT escaped: Typst's
// smart-dash substitution is a presentational concern, not a security one,
// and the only fully-portable way to suppress it is to insert non-empty
// content between hyphens, which changes the rendered glyph stream. If
// real data ever needs literal `--`, revisit.
func escapeTypstContent(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 8)
	for _, r := range s {
		switch r {
		// '/' is a metachar, not punctuation: "//" opens a line comment,
		// "/*" a block comment, and "/ " at line start a term list. Left
		// unescaped, "3/*4 of peak*/ sustained" silently renders as
		// "3 sustained" - the text between the delimiters just disappears.
		case '\\', '"', '*', '_', '`', '#', '<', '>', '@', '=',
			'[', ']', '(', ')', '~', '$', '/':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// escapeTypstString escapes a value for embedding inside a Typst `"..."`
// string literal. Only `\` and `"` are structural inside a string literal;
// every other char is fine. Used for URLs in #link("...").
func escapeTypstString(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	for _, r := range s {
		switch r {
		case '\\', '"':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
