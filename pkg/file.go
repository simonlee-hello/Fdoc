package pkg

import (
	"Fdoc/pkg/option"
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func FindFilesWithExtensions(rootDir string, extensions []string, skipDirs []string, dateLimit bool) ([]string, error) {
	var Files []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// 捕获权限错误并跳过该目录
			if os.IsPermission(err) {
				//fmt.Printf("权限不足，跳过目录：%s\n", path)
				return nil
			}
			return err
		}

		if info.IsDir() {
			// 检查目录是否需要跳过
			skip := false
			for _, skipDir := range skipDirs {
				if strings.Contains(path, skipDir) {
					skip = true
					break
				}
			}
			if skip {
				return filepath.SkipDir
			}
		} else {
			// 检查文件扩展名是否匹配所需的扩展名
			for _, ext := range extensions {
				if strings.EqualFold(filepath.Ext(info.Name()), ext) {
					// 如果有日期限制
					if dateLimit {
						// 获取文件的修改时间
						modTime := info.ModTime()

						// 检查修改时间是否在指定日期之后
						if modTime.After(option.AfterDate) {
							Files = append(Files, path)
						}
						break
					}
					Files = append(Files, path)
					break
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return Files, nil
}

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
func FilesToZip(files []string) {
	// 创建一个输出 ZIP 文件
	zipFile, err := os.Create(option.ZipPath)
	if err != nil {
		fmt.Println("无法创建 ZIP 文件:", err)
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
			fmt.Printf("无法打开文件 %s: %v\n", filePath, err)
			continue
		}
		defer file.Close()

		// 创建 ZIP 文件中的文件
		relPath, _ := filepath.Rel(option.RootDir, filePath)
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			fmt.Printf("无法创建 ZIP 文件中的文件 %s: %v\n", relPath, err)
			continue
		}

		// 将文件内容拷贝到 ZIP 文件中
		_, err = io.Copy(writer, file)
		if err != nil {
			fmt.Printf("无法拷贝文件 %s 到 ZIP 文件: %v\n", relPath, err)
			continue
		}
	}
	fmt.Printf("文件已成功打包到: %v", option.ZipPath)
}
