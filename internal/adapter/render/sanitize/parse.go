package sanitize

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// errNotLink is a soft-fail signal: the input doesn't match the link or
// #link() shape, but the leading character should be treated as literal
// text rather than as a parse error. Hard failures (e.g. shape matches
// but the URL is on a disallowed scheme) surface as ordinary errors.
var errNotLink = errors.New("not a link")

// parse converts input into a list of spans according to the DESIGN §3.3
// allowlist grammar. Composite spans (Bold/Italic/Link) keep their inner
// raw content in Span.Text; the emitter re-parses that recursively, which
// is how `*\`<100 ms\`*` round-trips correctly.
func parse(input string) ([]domain.Span, error) {
	if strings.ContainsRune(input, 0) {
		return nil, fmt.Errorf("input contains NUL byte")
	}
	if !utf8.ValidString(input) {
		return nil, fmt.Errorf("input is not valid UTF-8")
	}

	var spans []domain.Span
	var literal strings.Builder
	flush := func() {
		if literal.Len() > 0 {
			spans = append(spans, domain.Span{Kind: domain.SpanText, Text: literal.String()})
			literal.Reset()
		}
	}

	const legacyLinkPrefix = `#link("`

	for i := 0; i < len(input); {
		if strings.HasPrefix(input[i:], legacyLinkPrefix) {
			n, span, err := parseLegacyLink(input[i:])
			if err != nil {
				return nil, err
			}
			flush()
			spans = append(spans, span)
			i += n
			continue
		}

		c := input[i]
		switch c {
		case '*':
			n, span, err := parseDelimited(input[i:], '*', domain.SpanBold)
			if err != nil {
				return nil, err
			}
			flush()
			spans = append(spans, span)
			i += n
		case '_':
			n, span, err := parseDelimited(input[i:], '_', domain.SpanItalic)
			if err != nil {
				return nil, err
			}
			flush()
			spans = append(spans, span)
			i += n
		case '`':
			n, span, err := parseDelimited(input[i:], '`', domain.SpanCode)
			if err != nil {
				return nil, err
			}
			flush()
			spans = append(spans, span)
			i += n
		case '[':
			n, span, err := parseLink(input[i:])
			if errors.Is(err, errNotLink) {
				literal.WriteByte('[')
				i++
				continue
			}
			if err != nil {
				return nil, err
			}
			flush()
			spans = append(spans, span)
			i += n
		default:
			literal.WriteByte(c)
			i++
		}
	}
	flush()
	return spans, nil
}

// parseDelimited matches `<d>X<d>` with no `<d>` in X. Returns (bytes consumed, span, error).
func parseDelimited(s string, d byte, kind domain.SpanKind) (int, domain.Span, error) {
	if len(s) < 2 || s[0] != d {
		return 0, domain.Span{}, fmt.Errorf("expected %q", d)
	}
	rest := s[1:]
	end := strings.IndexByte(rest, d)
	if end < 0 {
		return 0, domain.Span{}, fmt.Errorf("unterminated %q delimiter", d)
	}
	inner := rest[:end]
	if inner == "" {
		return 0, domain.Span{}, fmt.Errorf("empty %q span", d)
	}
	return 1 + end + 1, domain.Span{Kind: kind, Text: inner}, nil
}

// parseLink matches `[text](url)`. Shape mismatches return errNotLink so
// the caller can fall back to literal `[`. URL validation failures are
// hard errors.
func parseLink(s string) (int, domain.Span, error) {
	if len(s) < 4 || s[0] != '[' {
		return 0, domain.Span{}, errNotLink
	}
	closeBracket := strings.IndexByte(s, ']')
	if closeBracket < 0 || closeBracket == 1 {
		return 0, domain.Span{}, errNotLink
	}
	if closeBracket+1 >= len(s) || s[closeBracket+1] != '(' {
		return 0, domain.Span{}, errNotLink
	}
	urlStart := closeBracket + 2
	closeParen := strings.IndexByte(s[urlStart:], ')')
	if closeParen < 0 {
		return 0, domain.Span{}, errNotLink
	}
	rawURL := s[urlStart : urlStart+closeParen]
	if err := validateURL(rawURL); err != nil {
		return 0, domain.Span{}, fmt.Errorf("link [text](url): %w", err)
	}
	return urlStart + closeParen + 1, domain.Span{
		Kind: domain.SpanLink,
		Text: s[1:closeBracket],
		URL:  rawURL,
	}, nil
}

// parseLegacyLink matches `#link("url")[text]` (v1 syntax). Anything that
// passes the `#link("` prefix check but doesn't match the full shape is
// a hard error — by the time we see that prefix, the user clearly
// intended a link.
func parseLegacyLink(s string) (int, domain.Span, error) {
	const prefix = `#link("`
	if !strings.HasPrefix(s, prefix) {
		return 0, domain.Span{}, fmt.Errorf("not a #link()")
	}
	rest := s[len(prefix):]
	closeQuote := strings.IndexByte(rest, '"')
	if closeQuote < 0 {
		return 0, domain.Span{}, fmt.Errorf(`legacy link: unterminated " in URL`)
	}
	rawURL := rest[:closeQuote]
	rest = rest[closeQuote+1:]
	if !strings.HasPrefix(rest, ")[") {
		return 0, domain.Span{}, fmt.Errorf(`legacy link: expected ")[" after URL`)
	}
	rest = rest[2:]
	closeBracket := strings.IndexByte(rest, ']')
	if closeBracket < 0 {
		return 0, domain.Span{}, fmt.Errorf("legacy link: unterminated text bracket")
	}
	text := rest[:closeBracket]
	if err := validateURL(rawURL); err != nil {
		return 0, domain.Span{}, fmt.Errorf(`legacy link #link("%s"): %w`, rawURL, err)
	}
	// len(prefix) + URL + closing " + ")[" + text + ]
	consumed := len(prefix) + closeQuote + 1 + 2 + closeBracket + 1
	return consumed, domain.Span{Kind: domain.SpanLink, Text: text, URL: rawURL}, nil
}
