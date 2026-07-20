package methods

import (
	"io"
)

type MultiPartUploadConfig struct {
	Debug      bool
	Endpoint   string
	FileName   string
	FileReader io.Reader
	FileSize   int64
	Headers    map[string]string
}

type TransferConfig struct {
	Parallel      int
	DebugMode     *bool
	NoBarMode     bool
	CryptoMode    bool
	CryptoKey     string
	MaxBytes      int64  // 0 = unlimited
	BackendName   string // for size-check messages
	RecursiveDirs bool   // true: upload each file under dir; false: zip dir then upload
}
