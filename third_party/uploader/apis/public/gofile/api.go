package gofile

import "uploader/apis"

var Backend = new(goFile)

type goFile struct {
	apis.Backend
	totalSize int64

	userToken  string
	serverLink string
	folderID   string
	folderName string

	downloadLink string

	Config   goFileOptions
	Commands [][]string
}
