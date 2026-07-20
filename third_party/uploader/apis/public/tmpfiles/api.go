package tempfiles

import "uploader/apis"

var Backend = new(tempf)

type tempf struct {
	apis.Backend
	resp     string
	Commands [][]string
}
