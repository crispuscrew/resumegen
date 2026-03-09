package stage

import (
	"github.com/crispuscrew/resumegen/internal/model"

	"errors"
)

var ErrRerun = errors.New("rerun")

func Execute(oldModel model.Model, stageExecute func(model.Model) (model.Model, error)) (model.Model, error) {
	var err error
	var newModel model.Model
	for {
		newModel, err = stageExecute(oldModel)
		if err == nil { return newModel, nil }
		if err == ErrRerun { continue }
		return oldModel, err
	}
}