package null

import "uploader/apis"

var Backend = new(null)

type null struct {
	apis.Backend
	resp     string
	Commands [][]string
}
