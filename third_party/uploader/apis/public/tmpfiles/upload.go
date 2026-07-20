package tempfiles

import (
	"encoding/json"
	"fmt"
	"io"
	"uploader/apis"
	"uploader/apis/methods"
)

const upload = "https://tmpfiles.org/api/v1/upload"

func (b *tempf) DoUpload(name string, size int64, file io.Reader) error {

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

	var resp uploadResp
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return fmt.Errorf("unmarshal returns error: %s", err)
	}
	if resp.Status != "success" {
		return fmt.Errorf("upload returns error: %s", string(body))
	}
	b.resp = resp.Data.URL

	return nil

}

func (b *tempf) PostUpload(string, int64) (string, error) {
	fmt.Println(b.resp)
	return b.resp, nil
}
