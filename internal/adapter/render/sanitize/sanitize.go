// Package sanitize closes the Typst code-injection vector by parsing
// bullet-text markup through a fixed allowlist grammar (DESIGN §3.3) and
// re-emitting every span as escaped Typst (§4.2 step 3). It is the
// always-on layer between user-supplied i18n strings and the generated
// Typst source.
package sanitize

// Mode controls what happens when input doesn't parse or contains an
// invalid URL scheme.
type Mode int

const (
	// Strict surfaces sanitizer errors to the caller (the render fails).
	Strict Mode = iota

	// Permissive emits the offending input as Typst-escaped literal text
	// (no formatting), so a single bad bullet doesn't fail the render.
	// The whole input is escaped — partial formatting recovery is not a
	// goal because security wins over UX here.
	Permissive
)

// Sanitize returns input as escaped Typst markup suitable for embedding
// inside a content block `[...]`.
func Sanitize(input string, mode Mode) (string, error) {
	spans, err := parse(input)
	if err != nil {
		if mode == Permissive {
			return escapeTypstContent(input), nil
		}
		return "", err
	}
	out, err := emit(spans)
	if err != nil {
		if mode == Permissive {
			return escapeTypstContent(input), nil
		}
		return "", err
	}
	return out, nil
}
