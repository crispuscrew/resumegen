package usecase_test

import (
	"testing"

	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

func TestTrimIsNeeded(t *testing.T) {
	cases := []struct {
		name    string
		pages   float64
		max     float64
		want    bool
		wantErr bool
	}{
		{"zero max disables", 5.0, 0, false, false},
		{"under limit", 0.9, 1.0, false, false},
		{"at limit", 1.0, 1.0, false, false},
		{"over limit", 1.1, 1.0, true, false},
		{"negative max errors", 1.0, -1, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := usecase.TrimIsNeeded(tc.pages, tc.max)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestTrimLowest_MarksLowestScoredBullet(t *testing.T) {
	data := domain.ResumeData{
		Jobs: []domain.Job{{
			Meta: domain.Meta{Score: 10, Reason: domain.Included},
			Bullets: []domain.Bullet{
				{Meta: domain.Meta{Score: 5, Reason: domain.Included}},
				{Meta: domain.Meta{Score: 2, Reason: domain.Included}},
				{Meta: domain.Meta{Score: 8, Reason: domain.Included}},
			},
		}},
	}
	out, trimmed := usecase.TrimLowest(data, domain.MinElements{JobBullets: 1})
	if !trimmed {
		t.Fatal("expected trim to take effect")
	}
	if out.Jobs[0].Bullets[1].Reason != domain.Trimmed {
		t.Errorf("idx-1 bullet (score=2) should be Trimmed, got %v", out.Jobs[0].Bullets[1].Reason)
	}
	if out.Jobs[0].Reason != domain.Included {
		t.Error("job should stay Included after a single bullet trim with min=1")
	}
}

func TestTrimLowest_PropagatesToSkillCatBelowMin(t *testing.T) {
	// The single lowest-scored nested element is the skill item (score=1),
	// so it gets trimmed. The cat then has 0 included items, falling below
	// SkillItems=1, so the cat itself becomes Trimmed.
	data := domain.ResumeData{
		Jobs: []domain.Job{{
			Meta: domain.Meta{Reason: domain.Included},
			Bullets: []domain.Bullet{
				{Meta: domain.Meta{Score: 10, Reason: domain.Included}},
			},
		}},
		SkillCats: []domain.SkillCat{{
			Meta: domain.Meta{Reason: domain.Included},
			Items: []domain.SkillItem{
				{Meta: domain.Meta{Score: 1, Reason: domain.Included}},
			},
		}},
	}
	out, _ := usecase.TrimLowest(data, domain.MinElements{
		JobBullets: 1, ProjectBullets: 1, SkillItems: 1,
	})
	if out.SkillCats[0].Reason != domain.Trimmed {
		t.Errorf("SkillCat should be Trimmed after its only item is trimmed, got %v", out.SkillCats[0].Reason)
	}
}

func TestTrimLowest_AllProjectBulletsTrimmedTrimsProject(t *testing.T) {
	data := domain.ResumeData{
		Projects: []domain.Project{{
			Meta: domain.Meta{Reason: domain.Included},
			Bullets: []domain.Bullet{
				{Meta: domain.Meta{Score: 2, Reason: domain.Included}},
			},
		}},
	}
	out, _ := usecase.TrimLowest(data, domain.MinElements{ProjectBullets: 1})
	if out.Projects[0].Reason != domain.Trimmed {
		t.Errorf("project with 0 included bullets after trim should be Trimmed, got %v", out.Projects[0].Reason)
	}
}

func TestTrimLowest_NothingToTrim(t *testing.T) {
	data := domain.ResumeData{}
	if _, trimmed := usecase.TrimLowest(data, domain.MinElements{}); trimmed {
		t.Error("empty data should not report trimmed=true")
	}
}
