package domain_test

import (
	"testing"

	"github.com/crispuscrew/resumegen/internal/domain"
)

func sampleScored() domain.ResumeData {
	return domain.ResumeData{
		Header: domain.Header{Name: domain.I18n{"en": "Ada"}},
		Jobs: []domain.Job{
			{
				Meta:    domain.Meta{Reason: domain.Included},
				Title:   domain.I18n{"en": "Engineer"},
				Company: domain.I18n{"en": "Acme"},
				Bullets: []domain.Bullet{
					{Meta: domain.Meta{Reason: domain.Included}, Text: domain.I18n{"en": "kept"}},
					{Meta: domain.Meta{Reason: domain.Trimmed}, Text: domain.I18n{"en": "dropped"}},
				},
			},
			{Meta: domain.Meta{Reason: domain.Filtered}, Company: domain.I18n{"en": "Hidden"}},
		},
		Projects: []domain.Project{
			{Meta: domain.Meta{Reason: domain.Trimmed}, Title: domain.I18n{"en": "Gone"}},
		},
		SkillCats: []domain.SkillCat{
			{
				Meta: domain.Meta{Reason: domain.Included},
				Name: domain.I18n{"en": "Languages"},
				Items: []domain.SkillItem{
					{Meta: domain.Meta{Reason: domain.Included}, Name: domain.I18n{"en": "Go"}},
					{Meta: domain.Meta{Reason: domain.Filtered}, Name: domain.I18n{"en": "COBOL"}},
				},
			},
		},
		Edu: []domain.Edu{{Title: domain.I18n{"en": "MIT"}}},
	}
}

func TestVisibleResume_DropsNonIncluded(t *testing.T) {
	v := domain.VisibleResume(sampleScored())

	if len(v.Jobs) != 1 || v.Jobs[0].Company.Lang("en") != "Acme" {
		t.Fatalf("want only the Included job, got %+v", v.Jobs)
	}
	if len(v.Jobs[0].Bullets) != 1 || v.Jobs[0].Bullets[0].Text.Lang("en") != "kept" {
		t.Errorf("want only the Included bullet, got %+v", v.Jobs[0].Bullets)
	}
	if len(v.Projects) != 0 {
		t.Errorf("trimmed project should be dropped, got %+v", v.Projects)
	}
	if len(v.SkillCats) != 1 || len(v.SkillCats[0].Items) != 1 || v.SkillCats[0].Items[0].Name.Lang("en") != "Go" {
		t.Errorf("want only the Included skill item, got %+v", v.SkillCats)
	}
}

func TestVisibleResume_KeepsEduAndHeader(t *testing.T) {
	v := domain.VisibleResume(sampleScored())
	if v.Header.Name.Lang("en") != "Ada" {
		t.Errorf("header should pass through unfiltered")
	}
	if len(v.Edu) != 1 {
		t.Errorf("education is always shown in full, got %+v", v.Edu)
	}
}
