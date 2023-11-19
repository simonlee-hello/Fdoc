package option

import (
	"flag"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"os"
	"os/user"
	"runtime"
)

type FlagInfo struct {
	MaxSize      string
	OutputPath   string
	AfterDateStr string
	RootPath     string
	SkipDirs     string
	FileName     string
	Keyword      string
	Extension    string
	Size         bool
}

func (info *FlagInfo) String() string {
	return fmt.Sprintf("MaxSize: \"%s\";"+
		"OutputPath: \"%s\";"+
		"AfterDateStr: \"%s\";"+
		"RootPath: \"%s\";"+
		"SkipDirs: \"%s\";"+
		"FileName: \"%s\";"+
		"Keyword: \"%s\";"+
		"Extension: \"%s\";"+
		"Size: %t",
		info.MaxSize, info.OutputPath, info.AfterDateStr, info.RootPath, info.SkipDirs, info.FileName, info.Keyword, info.Extension, info.Size)
}

func (info *FlagInfo) InitFlag() {
	info.GetFlag()
	// 初始化RootPath
	if info.RootPath == "" {
		// 获取当前用户的信息
		currentUser, err := user.Current()
		if err != nil {
			gologger.Warning().Msgf("Unable to obtain current user information:%v\n", err)
		}
		// 获取家目录
		homeDir := currentUser.HomeDir
		gologger.Debug().Msgf("current user home path : %s\n", homeDir)

		switch runtime.GOOS {
		case "windows":
			info.RootPath = homeDir
			if info.SkipDirs == "" {
				info.SkipDirs = "C:\\Windows, C:\\Program Files, C:\\Program Files (x86), C:\\inetpub, C:\\Users\\Public"
			}
		case "linux", "darwin":
			info.RootPath = homeDir

		}
	}
	// 检查目录是否存在
	_, dirErr := os.Stat(info.RootPath)
	if dirErr != nil {
		gologger.Error().Str("dir", info.RootPath).Msg("Directory does not exist!")
		os.Exit(0)
	}
	// 先判断输出文件路径是否存在
	_, OutputPathExistErr := os.Stat(info.OutputPath)
	if OutputPathExistErr == nil {
		gologger.Error().Str("tarGzPath", info.OutputPath).Msg("output tar.gz file exists, please rename tarGzPath")
		os.Exit(0)
	}
	gologger.Debug().Str("MaxSize", info.MaxSize).Msg("")
	gologger.Debug().Str("OutputPath", info.OutputPath).Msg("")
	gologger.Debug().Str("AfterDateStr", info.AfterDateStr).Msg("")
	gologger.Debug().Str("RootPath", info.RootPath).Msg("")
	gologger.Debug().Str("SkipDirs", info.SkipDirs).Msg("")
	gologger.Debug().Str("FileName", info.FileName).Msg("")
	gologger.Debug().Str("Keyword", info.Keyword).Msg("")
	gologger.Debug().Str("Extension", info.Extension).Msg("")
}

func (info *FlagInfo) GetFlag() {

	//flag.StringVar(&info.MaxSize, "maxSize", "1GB", "max file size can be zip")
	flag.StringVar(&info.MaxSize, "max", "1GB", "max file size can be zip (global option)")
	//flag.StringVar(&info.OutputPath, "zipPath", "output.zip", "zip output path")
	flag.StringVar(&info.OutputPath, "o", "output.tar.gz", "zip output path (global option)")
	//flag.StringVar(&info.AfterDateStr, "afterDateStr", "", "only query and pack the \"AfterDate\" file,Date in the format '2006-01-02'")
	flag.StringVar(&info.AfterDateStr, "t", "", "only query and pack files after the date,like '2023-10-01' (global option)(default \"\")")
	//flag.StringVar(&info.RootPath, "rootPath", "c:\\", "root path to query")
	flag.StringVar(&info.RootPath, "d", "", "root path to query (global option)")
	//flag.StringVar(&info.SkipDirs, "skipDirs", "C:\\Windows, C:\\Program Files, C:\\Program Files (x86), C:\\inetpub, C:\\Users\\Public", "paths to skip query")
	flag.StringVar(&info.SkipDirs, "x", "", "paths to skip query (global option)")
	flag.StringVar(&info.FileName, "f", "", "query files by filename (only for QueryByFileName),eg. '-f config  -f config,password,secret'")
	flag.StringVar(&info.Keyword, "k", "", "query files in content by keyword (only for QueryByKeyword),eg. '-k config -k password:,secret:,token:'")
	flag.StringVar(&info.Extension, "e", "", "query files by extension,eg. '-e pdf,doc,zip'")
	flag.BoolVar(&info.Size, "size", false, "Calculate total size")

	flag.Parse()
}
