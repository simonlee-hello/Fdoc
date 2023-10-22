package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// 计算文件总大小
func GetTotalSize(files []string) int64 {
	totalSize := int64(0)
	for _, filePath := range files {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			fmt.Printf("无法获取文件信息 %s: %v\n", filePath, err)
			continue
		}
		totalSize += fileInfo.Size()
	}
	return totalSize
}

// 将多个文件（files []string）打包到一个zip包中
func FilesToZip(rootDir string, zipPath string, files []string) error {
	// 创建一个输出 ZIP 文件
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	// 使用 zip.NewWriter 创建 ZIP 写入器
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 遍历文件列表并将它们添加到 ZIP 文件中
	for _, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("无法打开文件 %s: %v\n", filePath, err)
			continue
		}
		defer file.Close()

		// 获取文件信息
		info, err := file.Stat()
		if err != nil {
			fmt.Printf("获取文件信息失败: %v\n", err)
		}
		// 创建一个文件头，以保留日期等信息
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			fmt.Printf("创建文件头失败: %v\n", err)
		}

		// 创建 ZIP 文件中的文件
		relPath, _ := filepath.Rel(rootDir, filePath)
		header.Name = relPath
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// 将文件内容拷贝到 ZIP 文件中
		_, err = io.Copy(writer, file)
		if err != nil {
			fmt.Printf("无法拷贝文件 %s 到 ZIP 文件: %v\n", header.Name, err)
			continue
		}
	}
	fmt.Printf("文件已成功打包到: %v", zipPath)
	return nil
}
