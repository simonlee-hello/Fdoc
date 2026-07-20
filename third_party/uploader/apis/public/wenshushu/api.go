package wenshushu

import "uploader/apis"

var Backend = new(wssTransfer)

type wssTransfer struct {
	apis.Backend
	baseConf sendConfigBlock
	Config   wssOptions
	Commands [][]string
}
