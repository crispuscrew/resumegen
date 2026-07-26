package usecase_test

import (
	"testing"

	"github.com/crispuscrew/resumegen/internal/domain"
	"github.com/crispuscrew/resumegen/internal/usecase"
)

func TestScore_MatchingTagsBumpScore(t *testing.T) {
	data := domain.ResumeData{
		Jobs: []domain.Job{
			{Meta: domain.Meta{Tags: []string{"go", "backend"}}},
			{Meta: domain.Meta{Tags: []string{"frontend"}}},
			{Meta: domain.Meta{Tags: nil}},
		},
	}
	out := usecase.Score(data, []string{"go", "backend"}, domain.Score{})

	if out.Jobs[0].Score == 0 {
		t.Errorf("matching job score should be > 0, got 0")
	}
	if out.Jobs[0].Reason != domain.Included {
		t.Errorf("matching job Reason=%v, want Included", out.Jobs[0].Reason)
	}
	if out.Jobs[1].Reason != domain.Filtered {
		t.Errorf("non-matching tagged job Reason=%v, want Filtered", out.Jobs[1].Reason)
	}
	if out.Jobs[2].Reason != domain.Included {
		t.Errorf("untagged job Reason=%v, want Included (no tags = no filter)", out.Jobs[2].Reason)
	}
}

func TestScore_HighestPriorityTagWins(t *testing.T) {
	data := domain.ResumeData{
		Jobs: []domain.Job{
			{Meta: domain.Meta{Tags: []string{"go"}}},
			{Meta: domain.Meta{Tags: []string{"devops"}}},
		},
	}
	out := usecase.Score(data, []string{"go", "backend", "devops"}, domain.Score{})
	if out.Jobs[0].Score <= out.Jobs[1].Score {
		t.Errorf("higher-priority tag should score higher: go=%d devops=%d", out.Jobs[0].Score, out.Jobs[1].Score)
	}
}

func TestScore_SkillPriorityBumpsItems(t *testing.T) {
	data := domain.ResumeData{
		SkillCats: []domain.SkillCat{{
			Meta: domain.Meta{Tags: []string{"go"}},
			Items: []domain.SkillItem{
				{Meta: domain.Meta{Tags: []string{"go"}}},
				{Meta: domain.Meta{Tags: nil}},
			},
		}},
	}
	out := usecase.Score(data, []string{"go"}, domain.Score{SkillPriority: 7})
	items := out.SkillCats[0].Items
	if items[0].Score < 7 {
		t.Errorf("matching item score=%d, want >= 7", items[0].Score)
	}
	if items[1].Score != 7 {
		t.Errorf("untagged item score=%d, want exactly 7 (priority bump only)", items[1].Score)
	}
}

func TestScore_NestedBulletsFiltered(t *testing.T) {
	data := domain.ResumeData{
		Jobs: []domain.Job{{
			Meta: domain.Meta{Tags: []string{"go"}},
			Bullets: []domain.Bullet{
				{Meta: domain.Meta{Tags: []string{"go"}}},
				{Meta: domain.Meta{Tags: []string{"cpp"}}},
			},
		}},
	}
	out := usecase.Score(data, []string{"go"}, domain.Score{})
	if out.Jobs[0].Bullets[0].Reason != domain.Included {
		t.Error("matching bullet should be Included")
	}
	if out.Jobs[0].Bullets[1].Reason != domain.Filtered {
		t.Error("non-matching bullet should be Filtered")
	}
}
