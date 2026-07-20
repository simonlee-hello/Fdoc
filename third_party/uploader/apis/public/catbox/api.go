package catbox

import "uploader/apis"

var Backend = new(catBox)

type catBox struct {
	apis.Backend
	resp     string
	Commands [][]string
}
