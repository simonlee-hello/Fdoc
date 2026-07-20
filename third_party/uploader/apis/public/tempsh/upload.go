package tempsh

import (
	"fmt"
	"io"
	"strings"

	"uploader/apis"
	"uploader/apis/methods"
)

const upload = "https://temp.sh/upload"

func (b *temp) DoUpload(name string, size int64, file io.Reader) error {

	body, err := methods.MultipartUpload(methods.MultiPartUploadConfig{
		FileSize:   size,
		FileName:   name,
		FileReader: file,
		Debug:      apis.DebugMode,
		Endpoint:   upload,
	})
	if err != nil {
		return fmt.Errorf("upload returns error: %s", err)
	}
	if !strings.HasPrefix(string(body), "http://") && !strings.HasPrefix(string(body), "https://") {
		return fmt.Errorf("upload error: %s", string(body))
	}
	b.resp = string(body)
	return nil

}

func (b *temp) PostUpload(string, int64) (string, error) {
	apis.EmitLink(b.resp)
	return b.resp, nil
}
