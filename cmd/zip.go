package cmd

import (
	"Fdoc/pkg"
	"Fdoc/utils"
	"fmt"
)

func QueryAndZip(info *pkg.FlagInfo) {

	var files []string
	var err error
	if info.FileName != "" {
		files, err = pkg.QueryFilesByName(info)
	} else if info.Keyword != "" {
		files, err = pkg.QueryFilesByKeyword(info)
	} else {
		files, err = pkg.QueryFilesByExtensions(info)
	}
	if err != nil {
		fmt.Println("发生错误:", err)
		return
	}
	totalSize := utils.GetTotalSize(files)

	if totalSize > utils.SizeToBytes(info.MaxSize) {
		fmt.Printf("文件总大小 %s 超过 %s 字节，不执行打包操作。\n", utils.BytesToSize(totalSize), info.MaxSize)
		return
	}

	utils.FilesToZip(info.RootPath, info.ZipPath, files)
}
