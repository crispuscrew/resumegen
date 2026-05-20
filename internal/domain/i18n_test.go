package domain_test

import (
	"testing"

	"github.com/crispuscrew/resumegen/internal/domain"
)

func TestI18n_Lang(t *testing.T) {
	m := domain.I18n{"en": "Hello", "ru": "Привет"}
	if got := m.Lang("en"); got != "Hello" {
		t.Errorf("en lookup: %q", got)
	}
	if got := m.Lang("ru"); got != "Привет" {
		t.Errorf("ru lookup: %q", got)
	}
	if got := m.Lang("de"); got != "Hello" {
		t.Errorf("missing lang should fall back to en, got %q", got)
	}

	empty := domain.I18n{}
	if got := empty.Lang("en"); got != "" {
		t.Errorf("empty map should return empty string, got %q", got)
	}
}

func TestI18n_Has(t *testing.T) {
	m := domain.I18n{"en": "x"}
	if !m.Has("en") {
		t.Error("Has(en) should be true")
	}
	if m.Has("ru") {
		t.Error("Has(ru) should be false")
	}
}
