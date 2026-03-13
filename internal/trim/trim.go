package trim

import (
	"github.com/crispuscrew/resumegen/internal/model"
)

func TrimLowest(data model.ResumeData) model.ResumeData {
	min := minIncluded(model.FlatNested(data))
	min.Reason = model.Trimmed
	return data
}

func minIncluded(metas []*model.Meta) *model.Meta {
	var min *model.Meta
	for _, meta := range metas {
		if meta.Reason != model.Included { continue }
		if min == nil || meta.Score < min.Score { min = meta}
	}
	return min
}