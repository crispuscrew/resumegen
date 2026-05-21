package sanitize

import (
	"fmt"
	"strings"

	"github.com/crispuscrew/resumegen/internal/domain"
)

// emit serializes spans as Typst source. Bold/Italic/Link recursively parse
// and emit their inner content so nested markup (e.g. code inside bold)
// round-trips correctly. Plain text and URL strings go through their
// respective Typst escape functions before reaching the output.
func emit(spans []domain.Span) (string, error) {
	var b strings.Builder
	for _, span := range spans {
		switch span.Kind {
		case domain.SpanText:
			b.WriteString(escapeTypstContent(span.Text))
		case domain.SpanBold:
			inner, err := emitInner(span.Text)
			if err != nil {
				return "", err
			}
			b.WriteByte('*')
			b.WriteString(inner)
			b.WriteByte('*')
		case domain.SpanItalic:
			inner, err := emitInner(span.Text)
			if err != nil {
				return "", err
			}
			b.WriteByte('_')
			b.WriteString(inner)
			b.WriteByte('_')
		case domain.SpanCode:
			// Parser guaranteed no '`' inside; emit as raw.
			b.WriteByte('`')
			b.WriteString(span.Text)
			b.WriteByte('`')
		case domain.SpanLink:
			inner, err := emitInner(span.Text)
			if err != nil {
				return "", err
			}
			b.WriteString(`#link("`)
			b.WriteString(escapeTypstString(span.URL))
			b.WriteString(`")[`)
			b.WriteString(inner)
			b.WriteByte(']')
		default:
			return "", fmt.Errorf("unknown span kind %d", span.Kind)
		}
	}
	return b.String(), nil
}

func emitInner(raw string) (string, error) {
	spans, err := parse(raw)
	if err != nil {
		return "", err
	}
	return emit(spans)
}
