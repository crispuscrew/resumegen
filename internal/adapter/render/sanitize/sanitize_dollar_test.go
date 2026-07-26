package sanitize

import (
	"strings"
	"testing"
)

// The fresh-eyes audit's finding: `$` opens Typst math mode and was never
// escaped, so "cut spend from $8k to $5k" typeset as math and an unbalanced $
// could break the compile. It must come out escaped like every other metachar.
func TestDollarIsEscaped(t *testing.T) {
	out, err := Sanitize("Cut infra spend from $8,000 to $5,000/mo", Strict)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.ReplaceAll(out, `\$`, ""), "$") {
		t.Fatalf("unescaped $ in %q", out)
	}
	if !strings.Contains(out, `\$8,000`) {
		t.Fatalf("want escaped dollars, got %q", out)
	}
}
