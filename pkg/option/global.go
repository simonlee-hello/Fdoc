package option

import "time"

var (
	Extensions = []string{".pdf", ".docx", ".doc", ".xlsx", ".xls", ".csv", ".pptx", ".ppt", ".zip", ".rar", ".7z", ".tar", ".gz", ".tgz"}
	//MaxSize    = int64(10 * 1024 * 1024 * 1024) //10GB
	MaxSize   = int64(100 * 1024 * 1024) //100MB
	ZipPath   = "output.zip"             //压缩包路径
	AfterDate = time.Date(2023, 10, 21, 0, 0, 0, 0, time.UTC)
)
