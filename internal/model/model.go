package model

import (
	"github.com/crispuscrew/resumegen/internal/types"

	"io/fs"
)

type Model struct {
	UserChoise 	func(msg string, defaultVal bool) (bool)

	AppDirPath	string
	ProfileName	string

	AppDirFs	fs.FS

	Cfg			types.Config
	Data		types.ResumeData
	Profile		types.Profile
}