package apis

import (
	"io"
	"net/http"

	"uploader/utils"
)

type BaseBackend interface {
	Uploader
	LinkMatcher(string) bool
}

type Uploader interface {
	InitUpload([]string, []int64) error
	PreUpload(string, int64) error
	DoUpload(string, int64, io.Reader) error
	PostUpload(string, int64) (string, error)
	FinishUpload([]string) (string, error)

	StartProgress(io.Reader, int64) io.Reader
	EndProgress()
}

type Backend struct {
	BaseBackend
	prog *utils.ProgressReader
}

func (b *Backend) StartProgress(stream io.Reader, size int64) io.Reader {
	b.prog = utils.NewProgressReader(stream, size)
	return b.prog
}

func (b *Backend) EndProgress() {
	if b.prog != nil {
		b.prog.Finish()
		b.prog = nil
	}
}

func (b Backend) InitUpload([]string, []int64) error { return nil }
func (b Backend) FinishUpload([]string) (string, error) { return "", nil }
func (b Backend) PreUpload(string, int64) error         { return nil }
func (b Backend) DoUpload(string, int64, io.Reader) error {
	panic("DoUpload not implemented")
}
func (b Backend) PostUpload(string, int64) (string, error) { return "", nil }
func (b Backend) LinkMatcher(string) bool                   { return false }

const DefaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

func AddHeaders(req *http.Request) {
	req.Header.Set("User-Agent", DefaultUA)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", req.Host)
	req.Header.Set("Referer", req.Host)
}
