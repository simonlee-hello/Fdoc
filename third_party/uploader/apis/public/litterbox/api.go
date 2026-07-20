package litterbox

import "uploader/apis"

var Backend = new(litterbox)

type litterbox struct {
	apis.Backend
	resp     string
	Commands [][]string
}
