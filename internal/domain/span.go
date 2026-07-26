package domain

// SpanKind discriminates the parsed-markup variants the sanitizer produces.
type SpanKind int

const (
	SpanText SpanKind = iota
	SpanBold
	SpanItalic
	SpanCode
	SpanLink
)

// Span is one node in the parsed-markup tree of a bullet's text. For
// SpanText and SpanCode, Text is the literal content. For SpanBold,
// SpanItalic, and SpanLink, Text holds the raw inner content that the
// emitter re-parses to produce nested formatting. URL is only populated
// for SpanLink, and is guaranteed to have already passed the URL allowlist
// check at parse time.
type Span struct {
	Kind SpanKind
	Text string
	URL  string
}
