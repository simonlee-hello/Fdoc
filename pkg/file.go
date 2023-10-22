package pkg

import (
	"Fdoc/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func QueryFilesByExtensions(info *FlagInfo) ([]string, error) {
	var extensions = []string{".pdf", ".docx", ".doc", ".xlsx", ".xls", ".csv",
		".pptx", ".ppt", ".zip", ".rar", ".7z", ".tar", ".gz", ".tgz"}
	var files []string
	var afterDate time.Time
	if info.AfterDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", info.AfterDateStr)
		if err != nil {
			return nil, err
		}
		afterDate = parsedDate
	}
	err := filepath.Walk(info.RootPath, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			// 捕获权限错误并跳过该目录
			if os.IsPermission(err) {
				return nil
			}
			return err
		}
		if fileInfo.IsDir() {
			// 检查目录是否需要跳过
			skip := false
			for _, skipDir := range utils.ConvertStringToList(info.SkipDirs) {
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
				if strings.EqualFold(filepath.Ext(fileInfo.Name()), ext) {
					// 如果有日期限制，检查修改时间是否在指定日期之后
					if afterDate.IsZero() || fileInfo.ModTime().After(afterDate) {
						files = append(files, path)
					}
					break
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	fmt.Println(files)
	return files, nil
}
