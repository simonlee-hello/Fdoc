package fichier

import (
	"io"
)

type uploadConfig struct {
	debug      bool
	fileName   string
	fileReader io.Reader
	fileSize   int64

	email     string
	password  string
	uploadURL string
	apiKey    string
	domainID  string
}

type uploadServerResponse struct {
	URL string `json:"url"`
	ID  string `json:"id"`
}
