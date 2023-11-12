package pkg

import (
	"Fdoc/option"
	"Fdoc/utils"
	"github.com/projectdiscovery/gologger"
)

func QueryAndZip(info *option.FlagInfo) {

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
		gologger.Warning().Msg("file not found")
		gologger.Info().Msg("exiting..")
		return
	}

	// 如果命令参数包含--size，则计算所有符合条件的文件总大小并返回
	if info.Size {
		// Execute the GetTotalSize function
		totalSize := utils.BytesToSize(utils.GetTotalSize(files))
		gologger.Info().Msgf("totalSize:%v", totalSize)
		gologger.Info().Msg("exiting..")
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
