package domain_test

import (
	"strings"
	"testing"

	"github.com/crispuscrew/resumegen/internal/domain"
)

func nameData(s string) domain.ResumeData {
	return domain.ResumeData{Header: domain.Header{Name: domain.I18n{"en": s}}}
}

func summaryData(s string) domain.ResumeData {
	return domain.ResumeData{Header: domain.Header{Summary: domain.I18n{"en": s}}}
}

func bulletData(s string) domain.ResumeData {
	return domain.ResumeData{Jobs: []domain.Job{{Bullets: []domain.Bullet{{Text: domain.I18n{"en": s}}}}}}
}

func hrefData(s string) domain.ResumeData {
	return domain.ResumeData{Header: domain.Header{Contacts: []domain.Contact{{Href: s}}}}
}

func TestValidateInput(t *testing.T) {
	long := func(n int) string { return strings.Repeat("a", n) }

	tests := []struct {
		name    string
		data    domain.ResumeData
		strict  bool
		limits  domain.Limits
		wantErr string // "" expects success; otherwise a substring of the error
	}{
		// NUL is rejected regardless of strict mode, and at any depth.
		{"nul non-strict", nameData("a\x00b"), false, domain.Limits{}, "NUL byte"},
		{"nul strict", nameData("a\x00b"), true, domain.Limits{}, "NUL byte"},
		{"nul in bullet", bulletData("x\x00y"), false, domain.Limits{}, "NUL byte"},

		// Non-strict tolerates everything else (v1.0 data loads unchanged).
		{"oversize non-strict ok", nameData(long(5000)), false, domain.Limits{}, ""},
		{"control non-strict ok", nameData("a\x07b"), false, domain.Limits{}, ""},
		{"bad utf8 non-strict ok", nameData("a\xffb"), false, domain.Limits{}, ""},

		// Strict mode enforces the extra rules.
		{"control strict", nameData("a\x07b"), true, domain.Limits{}, "control character"},
		{"carriage return strict", nameData("a\rb"), true, domain.Limits{}, "control character"},
		{"newline allowed strict", summaryData("line1\nline2"), true, domain.Limits{}, ""},
		{"tab allowed strict", summaryData("a\tb"), true, domain.Limits{}, ""},
		{"bad utf8 strict", nameData("a\xffb"), true, domain.Limits{}, "invalid UTF-8"},

		// Byte limits (defaults: short 256, bullet_text 4096, url_or_path 2048).
		{"short over default", nameData(long(257)), true, domain.Limits{}, "exceeds limit 256"},
		{"short at default ok", nameData(long(256)), true, domain.Limits{}, ""},
		{"bullet over default", bulletData(long(4097)), true, domain.Limits{}, "exceeds limit 4096"},
		{"bullet uses its own class", bulletData(long(1000)), true, domain.Limits{}, ""}, // 1000 > short(256) but bullets allow 4096
		{"href over default", hrefData(long(2049)), true, domain.Limits{}, "exceeds limit 2048"},

		// Custom limits honored; other classes still default.
		{"custom short limit", nameData(long(11)), true, domain.Limits{Short: 10}, "exceeds limit 10"},
		{"custom short ok", nameData(long(10)), true, domain.Limits{Short: 10}, ""},

		// Clean data passes either way.
		{"clean non-strict", bulletData("Built a *REST API* in Go"), false, domain.Limits{}, ""},
		{"clean strict", bulletData("Built a *REST API* in Go"), true, domain.Limits{}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := domain.ValidateInput(tc.data, tc.strict, tc.limits)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected success, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}
