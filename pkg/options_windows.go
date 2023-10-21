package pkg

var (
	RootDir    = "C:\\" // 设置根目录路径为C盘
	Extensions = []string{".pdf", ".docx", ".doc", ".xlsx", ".xls", ".csv", ".pptx", ".ppt", ".zip", ".rar", ".7z", ".tar", ".gz", ".tgz"}
	SkipDirs   = []string{"C:\\Windows", "C:\\Program Files", "C:\\Program Files (x86)", "C:\\inetpub", "C:\\Users\\Public"}
	MaxSize    = int64(10 * 1024 * 1024 * 1024) //10GB
	ZipPath    = "output.zip"
)
