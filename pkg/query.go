package pkg

import (
	"Fdoc/utils"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var files []string

func WalkDir(f fs.WalkDirFunc) ([]string, error) {
	err := filepath.WalkDir(Info.RootPath, f)
	if err != nil {
		return nil, fmt.Errorf("Error walking directory: %v", err)
	}

	return files, nil
}

// WalkDirFunc
func QueryFilesByExtensions(path string, d fs.DirEntry, err error) error {
	if err != nil {
		// 捕获权限错误并跳过该目录
		if os.IsPermission(err) {
			return nil
		}
		return err
	}
	if d.IsDir() && !utils.IsSymlink(path) {
		// 检查目录是否需要跳过
		skip := false
		for _, skipDir := range utils.ConvertStringToList(Info.SkipDirs) {
			if strings.Compare(path, skipDir) == 0 {
				skip = true
				break
			}
		}
		if skip {
			return filepath.SkipDir
		}
	} else {
		// 获取文件扩展名
		ext := filepath.Ext(d.Name())
		// 检查文件扩展名是否匹配所需的扩展名
		var extensions = map[string]struct{}{
			".pdf": {}, ".docx": {}, ".doc": {}, ".xlsx": {}, ".xls": {}, ".csv": {},
			".pptx": {}, ".ppt": {}, ".zip": {}, ".rar": {}, ".7z": {}, ".tar": {}, ".gz": {}, ".tgz": {},
		}
		if _, ok := extensions[ext]; ok {

			// 如果有日期限制，检查修改时间是否在指定日期之后
			fileInfo, _ := d.Info()
			err = dateFilter(fileInfo, path)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// WalkDirFunc
func QueryFilesByName(path string, d fs.DirEntry, err error) error {
	if err != nil {
		// 捕获权限错误并跳过该目录
		if os.IsPermission(err) {
			return nil
		}
		return err
	}
	// 判断是否是文件且文件名包含指定部分
	//if !fileInfo.IsDir() && strings.Contains(strings.ToLower(fileInfo.Name()), strings.ToLower(info.FileName)) {
	//	matchingFiles = append(matchingFiles, path)
	//}
	if !d.IsDir() && !utils.IsSymlink(path) {
		// 如果 info.FileName 包含逗号，则拆分成多个查询字符串
		queryStrings := strings.Split(Info.FileName, ",")
		// 遍历所有查询字符串，检查是否包含在文件名中
		for _, queryString := range queryStrings {
			if strings.Contains(strings.ToLower(d.Name()), strings.ToLower(queryString)) {
				// 如果有日期限制，检查修改时间是否在指定日期之后
				fileInfo, _ := d.Info()
				err = dateFilter(fileInfo, path)
				if err != nil {
					return err
				}
				break // 跳出内层循环，避免重复添加
			}
		}
	}
	return nil
}

// WalkDirFunc
// 速度有些慢，待优化 TODO:优化速度
func QueryFilesByKeyword(path string, d fs.DirEntry, err error) error {
	if err != nil {
		// 捕获权限错误并跳过该目录
		if os.IsPermission(err) {
			return nil
		}
		return err
	}
	if !d.IsDir() && !utils.IsSymlink(path) {
		content, err := utils.ReadFile(path)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			return nil
		}
		// 遍历所有查询字符串，检查是否包含在文件名中
		// 如果 info.Keyword 包含逗号，则拆分成多个查询字符串
		queryStrings := strings.Split(Info.Keyword, ",")
		for _, queryString := range queryStrings {
			if strings.Contains(strings.ToLower(content), strings.ToLower(queryString)) {
				// 如果有日期限制，检查修改时间是否在指定日期之后
				fileInfo, _ := d.Info()
				err = dateFilter(fileInfo, path)
				if err != nil {
					return err
				}
				break // 跳出内层循环，避免重复添加
			}
		}
	}
	return nil
}

// 如果有日期限制，检查修改时间是否在指定日期之后
func dateFilter(fileInfo fs.FileInfo, path string) error {
	var afterDate time.Time
	if Info.AfterDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", Info.AfterDateStr)
		if err != nil {
			return fmt.Errorf("Failed to parse after date: %v", err)
		}
		afterDate = parsedDate
	}
	if afterDate.IsZero() || fileInfo.ModTime().After(afterDate) {
		files = append(files, path)
	}
	return nil
}
