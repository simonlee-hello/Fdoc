package option

import (
	"flag"
)

type FlagInfo struct {
	MaxSize      string
	ZipPath      string
	AfterDateStr string
	RootPath     string
	SkipDirs     string
	FileName     string
	Keyword      string
}

func GetFlag() *FlagInfo {
	info := &FlagInfo{}
	//flag.StringVar(&info.MaxSize, "maxSize", "1GB", "max file size can be zip")
	flag.StringVar(&info.MaxSize, "max", "1GB", "max file size can be zip (global option)")
	//flag.StringVar(&info.ZipPath, "zipPath", "output.zip", "zip output path")
	flag.StringVar(&info.ZipPath, "o", "output.tar.gz", "zip output path (global option)")
	//flag.StringVar(&info.AfterDateStr, "afterDateStr", "", "only query and pack the \"AfterDate\" file,Date in the format '2006-01-02'")
	flag.StringVar(&info.AfterDateStr, "t", "", "only query and pack files after the date,like '2023-10-01' (global option)(default \"\")")
	//flag.StringVar(&info.RootPath, "rootPath", "c:\\", "root path to query")
	flag.StringVar(&info.RootPath, "d", "c:\\", "root path to query (global option)")
	//flag.StringVar(&info.SkipDirs, "skipDirs", "C:\\Windows, C:\\Program Files, C:\\Program Files (x86), C:\\inetpub, C:\\Users\\Public", "paths to skip query")
	flag.StringVar(&info.SkipDirs, "x", "C:\\Windows, C:\\Program Files, C:\\Program Files (x86), C:\\inetpub, C:\\Users\\Public", "paths to skip query (global option)")
	flag.StringVar(&info.FileName, "f", "", "query files by filename (only for QueryByFileName),eg. '-f config  -f config,password,secret'")
	flag.StringVar(&info.Keyword, "k", "", "query files in content by keyword (only for QueryByKeyword),eg. '-k config -k password:,secret:,token:'")

	flag.Parse()
	return info
}
