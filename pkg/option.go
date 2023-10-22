package pkg

import (
	"flag"
)

type FlagInfo struct {
	MaxSize      string
	ZipPath      string
	AfterDateStr string
	RootPath     string
	SkipDirs     string
}

func GetFlag() *FlagInfo {
	info := &FlagInfo{}
	//flag.StringVar(&info.MaxSize, "maxSize", "1GB", "max file size can be zip")
	flag.StringVar(&info.MaxSize, "max", "1GB", "max file size can be zip (short)")
	//flag.StringVar(&info.ZipPath, "zipPath", "output.zip", "zip output path")
	flag.StringVar(&info.ZipPath, "o", "output.zip", "zip output path (short)")
	//flag.StringVar(&info.AfterDateStr, "afterDateStr", "", "only query and pack the \"AfterDate\" file,Date in the format '2006-01-02'")
	flag.StringVar(&info.AfterDateStr, "t", "", "only query and pack the 'AfterDate' file, Date in the format '2006-01-02' (short)")
	//flag.StringVar(&info.RootPath, "rootPath", "c:\\", "root path to query")
	flag.StringVar(&info.RootPath, "d", "c:\\", "root path to query (short)")
	//flag.StringVar(&info.SkipDirs, "skipDirs", "C:\\Windows, C:\\Program Files, C:\\Program Files (x86), C:\\inetpub, C:\\Users\\Public", "paths to skip query")
	flag.StringVar(&info.SkipDirs, "x", "C:\\Windows, C:\\Program Files, C:\\Program Files (x86), C:\\inetpub, C:\\Users\\Public", "paths to skip query (short)")
	flag.Parse()
	return info
}
