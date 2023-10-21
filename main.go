package main

import (
	"Fdoc/pkg"
	"fmt"
)

func main() {

	pdfDocxFiles, err := pkg.FindFilesWithExtensions(pkg.RootDir, pkg.Extensions, pkg.SkipDirs)
	if err != nil {
		fmt.Println("发生错误:", err)
		return
	}

	totalSize := pkg.GetTotalSize(pdfDocxFiles)

	if totalSize > pkg.MaxSize {
		fmt.Printf("文件总大小 %d 字节超过 %d 字节，不执行打包操作。\n", totalSize, pkg.MaxSize)
		return
	}

	pkg.FilesToZip(pdfDocxFiles)
}
