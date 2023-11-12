package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
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

// 将多个文件（files []string）打包到一个tar+gzip归档中
func FilesToTarGz(rootDir string, tarGzPath string, files []string) {

	// 创建一个输出tar+gzip归档文件
	tarGzFile, err := os.Create(tarGzPath)
	if err != nil {
		gologger.Error().Msgf("output file create failed")
		return
	}
	defer tarGzFile.Close()

	// 创建一个gzip写入器
	//gzWriter := gzip.NewWriter(tarGzFile)
	gzWriter, _ := gzip.NewWriterLevel(tarGzFile, gzip.BestSpeed)

	defer gzWriter.Close()

	// 创建一个tar写入器
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// 遍历文件列表并将它们添加到tar归档中
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

		// 创建tar头
		header := new(tar.Header)
		header.Name, _ = filepath.Rel(rootDir, filePath)
		header.Name = TransformSlash(header.Name)
		header.Size = info.Size()
		header.Mode = int64(info.Mode())
		header.ModTime = info.ModTime()

		// 将头部写入tar归档
		if err := tarWriter.WriteHeader(header); err != nil {
			gologger.Warning().Msgf("Failed to write tar header: %v\n", err)
			continue
		}

		// 将文件内容拷贝到tar归档中
		_, err = io.Copy(tarWriter, file)
		if err != nil {
			gologger.Warning().Msgf("Unable to copy %s to tar archive: %v\n", header.Name, err)
			continue
		}
	}
	tarGzSize := BytesToSize(GetTotalSize([]string{tarGzPath}))

	gologger.Info().Str("path", tarGzPath).Str("size", tarGzSize).Msg("SUCCESS!")
}

func TransformSlash(input string) string {
	return strings.Replace(input, `\`, `/`, -1)
}
