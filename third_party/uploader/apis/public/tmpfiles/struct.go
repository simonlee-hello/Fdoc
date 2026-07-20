package tempfiles

import (
	"io"
)

type uploadConfig struct {
	debug      bool
	fileName   string
	fileReader io.Reader
	fileSize   int64
}

type uploadResp struct {
	Status string `json:"status"`
	Data   struct {
		URL string `json:"url"`
	} `json:"data"`
}
