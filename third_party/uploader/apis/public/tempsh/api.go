package tempsh

import "uploader/apis"

var Backend = new(temp)

type temp struct {
	apis.Backend
	resp     string
	Commands [][]string
}
