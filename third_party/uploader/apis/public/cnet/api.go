package cnet

import "uploader/apis"

var Backend = new(cnet)

type cnet struct {
	apis.Backend
	resp     string
	Commands [][]string
}
