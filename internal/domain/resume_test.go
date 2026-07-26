package domain_test

import (
	"testing"

	"github.com/crispuscrew/resumegen/internal/domain"
)

func TestFlatTopLevel(t *testing.T) {
	data := domain.ResumeData{
		Jobs:      []domain.Job{{}, {}},
		Projects:  []domain.Project{{}},
		SkillCats: []domain.SkillCat{{}, {}, {}},
	}
	got := domain.FlatTopLevel(data)
	if len(got) != 6 {
		t.Errorf("len=%d, want 6 (2 jobs + 1 project + 3 skillcats)", len(got))
	}
}

func TestFlatNested(t *testing.T) {
	data := domain.ResumeData{
		Jobs: []domain.Job{
			{Bullets: []domain.Bullet{{}, {}}},
		},
		Projects: []domain.Project{
			{Bullets: []domain.Bullet{{}}},
		},
		SkillCats: []domain.SkillCat{
			{Items: []domain.SkillItem{{}, {}, {}}},
		},
	}
	got := domain.FlatNested(data)
	if len(got) != 6 {
		t.Errorf("len=%d, want 6 (2 job bullets + 1 proj bullet + 3 skill items)", len(got))
	}
}

func TestMeta_GetMeta(t *testing.T) {
	j := &domain.Job{Meta: domain.Meta{Score: 42}}
	if j.GetMeta().Score != 42 {
		t.Errorf("embedded Meta accessor broken")
	}
}
