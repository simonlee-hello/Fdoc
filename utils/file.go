package utils

import (
	"archive/zip"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// 计算文件总大小
func GetTotalSize(files []string) int64 {
	totalSize := int64(0)
	for _, filePath := range files {
		fileInfo, err := os.Lstat(filePath)
		if err != nil {
			gologger.Error().Msgf("Unable to obtain file information %s: %v\n", filePath, err)
			continue
		}
		totalSize += fileInfo.Size()
	}
	return totalSize
}

func GetTotalSizeAndCheckLimit(files []string, maxSize int64) (int64, bool) {
	totalSize := int64(0)
	for _, filePath := range files {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			gologger.Warning().Str("filePath", filePath).Str("err", err.Error()).Msg("Unable to obtain file information.")
			continue
		}
		totalSize += fileInfo.Size()

		if totalSize > maxSize {
			return totalSize, true // 返回标志表示超过了预设值
		}
	}
	return totalSize, false // 返回标志表示未超过预设值
}

// 将多个文件（files []string）打包到一个zip包中
func FilesToZip(rootDir string, zipPath string, files []string) {
	// 创建一个输出 ZIP 文件
	zipFile, err := os.Create(zipPath)
	if err != nil {
		gologger.Error().Msgf("output file create failed")
		return
	}
	defer zipFile.Close()

	// 使用 zip.NewWriter 创建 ZIP 写入器
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 遍历文件列表并将它们添加到 ZIP 文件中
	for _, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			gologger.Warning().Msgf("Unable to open the file %s: %v\n", filePath, err)
			continue
		}
		defer file.Close()

		// 获取文件信息
		info, err := file.Stat()
		if err != nil {
			gologger.Warning().Msgf("Failed to obtain file information: %v\n", err)
			continue
		}
		// 创建一个文件头，以保留日期等信息
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			gologger.Warning().Msgf("Failed to read file header: %v\n", err)
			continue
		}

		// 创建 ZIP 文件中的文件
		relPath, _ := filepath.Rel(rootDir, filePath)
		header.Name = relPath
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			gologger.Warning().Msgf("Failed to create file header: %v\n", err)
			continue
		}

		// 将文件内容拷贝到 ZIP 文件中
		_, err = io.Copy(writer, file)
		if err != nil {
			gologger.Warning().Msgf("Unable to copy %s to ZIP file: %v\n", header.Name, err)
			continue
		}
	}
	gologger.Info().Msgf("The file has been successfully packaged to: %v", zipPath)
}

/*
读取文件内容
*/
func ReadFile(fileName string) (string, error) {
	b, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return "", err
	} else {
		content := string(b[:])
		return content, nil
	}
}

func IsSymlink(file string) bool {
	fi, err := os.Lstat(file)
	if err != nil {
		return false
	}
	mode := fi.Mode()
	return mode&os.ModeSymlink != 0
}

func TransformSlash(input string) string {
	return strings.Replace(input, `\`, `/`, -1)
}

//func IsTarGzEmpty(filename string, threshold int64) bool {
//	fileInfo, err := os.Stat(filename)
//	if err != nil {
//		gologger.Warning().Msgf("IsTarGzEmpty stat file failed, err: %v", err)
//		return false
//	}
//
//	// 检查文件大小是否小于阈值
//	return fileInfo.Size() <= threshold
//}

func DeleteFile(path string) {
	err := os.Remove(path)
	if err != nil {
		gologger.Error().Str("path", path).Msg("delete failed")
		return
	}
}
