package sanitize

import (
	"fmt"
	"net/url"
	"strings"
)

// allowedSchemes is the closed set per DESIGN §3.3. Anything else (file:,
// javascript:, data:, ssh://, etc.) is rejected at parse time.
var allowedSchemes = map[string]bool{
	"http":   true,
	"https":  true,
	"mailto": true,
}

// validateURL rejects empty input, control characters, malformed URLs, and
// any scheme outside the allowlist.
func validateURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("URL is empty")
	}
	if strings.ContainsAny(raw, "\x00\n\r\t") {
		return fmt.Errorf("URL contains control characters")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("URL is malformed: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if !allowedSchemes[scheme] {
		return fmt.Errorf("URL scheme %q is not allowed (only http, https, mailto)", u.Scheme)
	}
	return nil
}
