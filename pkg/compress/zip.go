package compress

import (
	"archive/zip"
	"github.com/projectdiscovery/gologger"
	"io"
	"os"
	"path/filepath"
)

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
