package pkg

var (
	RootDir    = "/Users/simon/Downloads" // 设置根目录路径为/
	Extensions = []string{".pdf", ".docx", ".doc", ".xlsx", ".xls", ".csv", ".pptx", ".ppt", ".zip", ".rar", ".7z", ".tar", ".gz", ".tgz"}
	SkipDirs   = []string{}
	//MaxSize    = int64(10 * 1024 * 1024 * 1024) //10GB
	MaxSize = int64(100 * 1024 * 1024) //100MB
	ZipPath = "output.zip"
)
