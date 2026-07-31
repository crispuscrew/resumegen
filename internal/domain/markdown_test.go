package domain_test

import (
	"strings"
	"testing"

	"github.com/crispuscrew/resumegen/internal/domain"
)

func TestRenderMarkdown_StructureAndFiltering(t *testing.T) {
	md := string(domain.RenderMarkdown(sampleScored(), domain.Profile{Lang: "en"}))

	mustContain(t, md, "# Ada")
	mustContain(t, md, "## Experience")
	mustContain(t, md, "### Engineer — Acme")
	mustContain(t, md, "- kept")
	mustContain(t, md, "## Skills")
	mustContain(t, md, "- Languages: Go")
	mustContain(t, md, "## Education")
	mustContain(t, md, "### MIT")

	if strings.Contains(md, "dropped") {
		t.Errorf("trimmed bullet leaked into markdown:\n%s", md)
	}
	if strings.Contains(md, "Hidden") || strings.Contains(md, "COBOL") || strings.Contains(md, "Gone") {
		t.Errorf("filtered entity leaked into markdown:\n%s", md)
	}
}

func TestRenderMarkdown_RawMarkupPreserved(t *testing.T) {
	data := domain.ResumeData{
		Jobs: []domain.Job{{
			Meta:    domain.Meta{Reason: domain.Included},
			Title:   domain.I18n{"en": "Engineer"},
			Bullets: []domain.Bullet{{Meta: domain.Meta{Reason: domain.Included}, Text: domain.I18n{"en": "Built a *REST API*"}}},
		}},
	}
	md := string(domain.RenderMarkdown(data, domain.Profile{Lang: "en"}))
	mustContain(t, md, "- Built a *REST API*")
}

func TestRenderMarkdown_LangProjection(t *testing.T) {
	data := domain.ResumeData{
		Header: domain.Header{Name: domain.I18n{"en": "Ada", "ru": "Ада"}},
	}
	md := string(domain.RenderMarkdown(data, domain.Profile{Lang: "ru"}))
	mustContain(t, md, "# Ада")
	if strings.Contains(md, "Ada") {
		t.Errorf("en projection leaked into ru output:\n%s", md)
	}
}

func mustContain(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Errorf("output missing %q:\n%s", sub, s)
	}
}
