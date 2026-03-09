package model

import (
	"io/fs"
)

type Model struct {
	UserChoise 	func(msg string, defaultVal bool) (bool)

	AppDirPath	string
	ProfileName	string

	AppDirFs	fs.FS

	Cfg			Config
	Data		ResumeData
	Profile		Profile
}