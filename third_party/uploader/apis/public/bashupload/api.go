package bashupload

import "uploader/apis"

var Backend = new(bash)

type bash struct {
	apis.Backend
	resp     string
	Commands [][]string
}
