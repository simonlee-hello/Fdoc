package pkg

import (
	"Fdoc/option"
	"Fdoc/utils"
	"fmt"
	"io/fs"
	"os"
)

var Info = option.GetFlag()

func QueryAndZip() {

	// 检查目录是否存在
	_, dirErr := os.Stat(Info.RootPath)
	if dirErr != nil {
		fmt.Errorf("Directory does not exist: %v", dirErr)
		return
	}

	var files []string
	var err error
	var queryFunc fs.WalkDirFunc
	if Info.FileName != "" {
		queryFunc = QueryFilesByName
	} else if Info.Keyword != "" {
		queryFunc = QueryFilesByKeyword
	} else {
		queryFunc = QueryFilesByExtensions
	}
	// 遍历并查询出所有files path
	files, err = WalkDir(queryFunc)

	if err != nil {
		fmt.Println("发生错误:", err)
		return
	}
	if files == nil {
		fmt.Println("file not found...\nexiting...")
		return
	}
	totalSize, limit := utils.GetTotalSizeAndCheckLimit(files, utils.SizeToBytes(Info.MaxSize))

	if limit {
		fmt.Printf("文件总大小 %s 超过 %s 字节，不执行打包操作。\n", utils.BytesToSize(totalSize), Info.MaxSize)
		return
	}

	utils.FilesToZip(Info.RootPath, Info.ZipPath, files)
}
