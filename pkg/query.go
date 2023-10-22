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

func QueryFilesByName(info *FlagInfo) ([]string, error) {
	var matchingFiles []string
	// 如果 info.FileName 包含逗号，则拆分成多个查询字符串
	queryStrings := strings.Split(info.FileName, ",")
	// 使用 filepath.Walk 遍历目录
	err := filepath.Walk(info.RootPath, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 判断是否是文件且文件名包含指定部分
		//if !fileInfo.IsDir() && strings.Contains(strings.ToLower(fileInfo.Name()), strings.ToLower(info.FileName)) {
		//	matchingFiles = append(matchingFiles, path)
		//}
		if !fileInfo.IsDir() {
			// 遍历所有查询字符串，检查是否包含在文件名中
			for _, queryString := range queryStrings {
				if strings.Contains(strings.ToLower(fileInfo.Name()), strings.ToLower(queryString)) {
					matchingFiles = append(matchingFiles, path)
					break // 跳出内层循环，避免重复添加
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error:", err)
		return nil, nil
	}
	return matchingFiles, nil
}

// 速度有些慢，待优化 TODO:优化速度
func QueryFilesByKeyword(info *FlagInfo) ([]string, error) {
	var matchingFiles []string
	// 如果 info.Keyword 包含逗号，则拆分成多个查询字符串
	queryStrings := strings.Split(info.Keyword, ",")
	// 使用 filepath.Walk 遍历目录
	err := filepath.Walk(info.RootPath, func(path string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fileInfo.IsDir() {
			content, err := utils.ReadFile(path)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				return nil
			}
			// 遍历所有查询字符串，检查是否包含在文件名中
			for _, queryString := range queryStrings {
				if strings.Contains(strings.ToLower(content), strings.ToLower(queryString)) {
					matchingFiles = append(matchingFiles, path)
					break // 跳出内层循环，避免重复添加
				}
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error:", err)
		return nil, nil
	}
	return matchingFiles, nil
}
