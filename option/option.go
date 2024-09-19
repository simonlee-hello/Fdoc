package option

import (
	"flag"
	"github.com/projectdiscovery/gologger"
	"os"
	"os/user"
	"runtime"
)

// FlagInfo 结构体定义了所有的命令行参数
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

// InitFlag 初始化命令行参数
func (info *FlagInfo) InitFlag() {
	info.GetFlag()
	info.initRootPath()
	info.checkDirectory(info.RootPath, "目录不存在!")
	info.checkFileExistence(info.OutputPath, "输出 tar.gz 文件已存在，请重命名 tarGzPath")
	info.logDebugInfo()
}

// GetFlag 获取命令行参数
func (info *FlagInfo) GetFlag() {
	flag.StringVar(&info.MaxSize, "max", "1GB", "最大文件大小 (全局选项)")
	flag.StringVar(&info.OutputPath, "o", "output.tar.gz", "压缩输出路径 (全局选项)")
	flag.StringVar(&info.AfterDateStr, "t", "", "仅查询并打包指定日期之后的文件，例如 '2023-10-01' (全局选项)(默认 \"\")")
	flag.StringVar(&info.RootPath, "d", "", "查询的根路径 (全局选项)")
	flag.StringVar(&info.SkipDirs, "x", "", "跳过查询的路径 (全局选项)")
	flag.StringVar(&info.FileName, "f", "", "按文件名查询文件 (仅用于 QueryByFileName)，例如 '-f config  -f config,password,secret'")
	flag.StringVar(&info.Keyword, "k", "", "按关键字查询文件内容 (仅用于 QueryByKeyword)，例如 '-k config -k password:,secret:,token:'")
	flag.StringVar(&info.Extension, "e", "", "按扩展名查询文件，例如 '-e pdf,doc,zip'")
	flag.BoolVar(&info.Size, "size", false, "计算总大小")
	flag.Parse()
}

// initRootPath 初始化根路径
func (info *FlagInfo) initRootPath() {
	if info.RootPath == "" {
		currentUser, err := user.Current()
		if err != nil {
			gologger.Warning().Msgf("无法获取当前用户信息:%v\n", err)
		}
		homeDir := currentUser.HomeDir
		gologger.Debug().Msgf("当前用户家目录 : %s\n", homeDir)

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
}

// checkDirectory 检查目录是否存在
func (info *FlagInfo) checkDirectory(path string, errMsg string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		gologger.Error().Str("dir", path).Msg(errMsg)
		os.Exit(0)
	}
}

// checkFileExistence 检查文件是否存在
func (info *FlagInfo) checkFileExistence(path string, errMsg string) {
	if _, err := os.Stat(path); err == nil {
		gologger.Error().Str("tarGzPath", path).Msg(errMsg)
		os.Exit(0)
	}
}

// logDebugInfo 记录调试信息
func (info *FlagInfo) logDebugInfo() {
	gologger.Debug().Str("MaxSize", info.MaxSize).Msg("")
	gologger.Debug().Str("OutputPath", info.OutputPath).Msg("")
	gologger.Debug().Str("AfterDateStr", info.AfterDateStr).Msg("")
	gologger.Debug().Str("RootPath", info.RootPath).Msg("")
	gologger.Debug().Str("SkipDirs", info.SkipDirs).Msg("")
	gologger.Debug().Str("FileName", info.FileName).Msg("")
	gologger.Debug().Str("Keyword", info.Keyword).Msg("")
	gologger.Debug().Str("Extension", info.Extension).Msg("")
}
