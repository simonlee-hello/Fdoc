package downloadgg

import "uploader/apis"

var Backend = new(dlg)

type dlg struct {
	apis.Backend
	resp     string
	Commands [][]string
}
