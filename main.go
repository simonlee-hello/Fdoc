package main

import (
	"Fdoc/pkg"
	"Fdoc/pkg/option"
	"fmt"
)

func main() {

	pdfDocxFiles, err := pkg.FindFilesWithExtensions(option.RootDir, option.Extensions, option.SkipDirs, false)
	if err != nil {
		fmt.Println("发生错误:", err)
		return
	}

	totalSize := pkg.GetTotalSize(pdfDocxFiles)

	if totalSize > option.MaxSize {
		fmt.Printf("文件总大小 %d 字节超过 %d 字节，不执行打包操作。\n", totalSize, option.MaxSize)
		return
	}

	pkg.FilesToZip(pdfDocxFiles)
}
