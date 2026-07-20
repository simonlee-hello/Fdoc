package bashupload

import (
	"fmt"
	"io"
	"regexp"
	"uploader/apis"
	"uploader/apis/methods"
)

const upload = "https://bashupload.com/"

func (b *bash) DoUpload(name string, size int64, file io.Reader) error {

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
	re := regexp.MustCompile(`wget (https://bashupload.com/.+)\n`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) != 2 {
		return fmt.Errorf("failed to parse upload response: %s", body)
	}
	b.resp = matches[1]
	return nil

}

func (b *bash) PostUpload(string, int64) (string, error) {
	fmt.Println(b.resp)
	return b.resp, nil
}
