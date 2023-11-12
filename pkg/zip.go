package pkg

import (
	"Fdoc/option"
	"Fdoc/utils"
	"github.com/projectdiscovery/gologger"
	"os"
)

func QueryAndZip() {
	info := &option.FlagInfo{}
	info.GetFlag()
	gologger.Debug().Str("MaxSize", info.MaxSize).Msg("")
	gologger.Debug().Str("OutputPath", info.OutputPath).Msg("")
	gologger.Debug().Str("AfterDateStr", info.AfterDateStr).Msg("")
	gologger.Debug().Str("RootPath", info.RootPath).Msg("")
	gologger.Debug().Str("SkipDirs", info.SkipDirs).Msg("")
	gologger.Debug().Str("FileName", info.FileName).Msg("")
	gologger.Debug().Str("Keyword", info.Keyword).Msg("")
	gologger.Debug().Str("Extension", info.Extension).Msg("")
	// 检查目录是否存在
	_, dirErr := os.Stat(info.RootPath)
	if dirErr != nil {
		gologger.Error().Str("dir", info.RootPath).Msg("Directory does not exist!")
		return
	}
	// 先判断输出文件路径是否存在
	_, OutputPathExistErr := os.Stat(info.OutputPath)
	if OutputPathExistErr == nil {
		gologger.Error().Str("tarGzPath", info.OutputPath).Msg("output tar.gz file exists, please rename tarGzPath")
		os.Exit(0)
	}

	var files []string
	var err error

	// 遍历并查询出所有files path
	rootPath := info.RootPath
	skipDIrs := info.SkipDirs
	files = WalkQuery(rootPath, skipDIrs, info)

	//fmt.Println(files)
	gologger.Verbose().Msgf("files: %v", files)

	if err != nil {
		gologger.Error().Str("err", err.Error()).Msg("WalkQuery function error")
		return
	}
	if files == nil {
		gologger.Warning().Msg("file not found...\nexiting...")
		return
	}

	// 如果命令参数包含--size，则计算所有符合条件的文件总大小并返回
	if info.Size {
		// Execute the GetTotalSize function
		totalSize := utils.BytesToSize(utils.GetTotalSize(files))
		gologger.Info().Msgf("totalSize: %v\nexiting..", totalSize)
		return
	}

	totalSize, limit := utils.GetTotalSizeAndCheckLimit(files, utils.SizeToBytes(info.MaxSize))
	if limit {
		//fmt.Printf("totalSize more than %s, exit!\n", info.MaxSize)
		gologger.Error().Msgf("totalSize more than %s, exit!\n", info.MaxSize)
		return
	}
	gologger.Info().Msgf("totalSize:%v\n", utils.BytesToSize(totalSize))

	//utils.FilesToZip(Info.RootPath, Info.OutputPath, files)
	utils.FilesToTarGz(info.RootPath, info.OutputPath, files)

}
