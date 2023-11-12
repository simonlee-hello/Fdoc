package pkg

import (
	"Fdoc/option"
	"Fdoc/utils"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func WalkQuery(rootPath string, skipDirs string, info *option.FlagInfo) []string {
	var files []string
	err := walkInternal(info.RootPath, skipDirs, info, &files)
	if err != nil {
		gologger.Error().Msgf("Error walking directory: %v", err)
		return nil
	}

	return files
}

func walkInternal(rootPath string, skipDirs string, info *option.FlagInfo, files *[]string) error {
	return filepath.WalkDir(
		rootPath,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// 捕获权限错误并跳过该目录
				if os.IsPermission(err) {
					gologger.Warning().Msgf("permission error: %v", err)
					return nil
				}
				return nil
			}

			// 检查目录是否需要跳过
			if d.IsDir() {
				skip := false
				for _, skipDir := range utils.ConvertStringToList(skipDirs) {
					if strings.Compare(path, skipDir) == 0 {
						skip = true
						break
					}
				}
				if skip {
					return filepath.SkipDir
				}

				// 过滤日期
			} else {
				// 修改此处，判断是否为符号链接，如果是，递归遍历链接目标
				if d.Type()&fs.ModeSymlink != 0 {
					linkTarget, err := os.Readlink(path)
					if err != nil {
						gologger.Error().Msgf("Error reading symlink: %v\n", err)
						return nil
					}
					// 获取绝对路径
					absLinkTarget := filepath.Join(filepath.Dir(path), linkTarget)
					gologger.Debug().Str("absLinkTarget", absLinkTarget)
					// Check if the absolute link target exists
					_, err = os.Stat(absLinkTarget)
					if err != nil {
						// Skip if the link target doesn't exist
						if os.IsNotExist(err) {
							gologger.Warning().Str("path", absLinkTarget).Msg("file not found")
							return nil
						}
						gologger.Error().Msgf("Error checking symlink target: %v\n", err)
						return nil
					}
					// 递归遍历链接目标
					return walkInternal(absLinkTarget, skipDirs, info, files)
				}

				// 不是符号链接，正常处理
				flagQuery(files, info, path, d)
			}

			return nil
		},
	)
}

// func WalkQuery(info *option.FlagInfo) ([]string, error) {
//func WalkQuery(rootPath string, skipDirs string, info *option.FlagInfo) ([]string, error) {
//	var files []string
//	err := filepath.WalkDir(
//		rootPath,
//		func(path string, d fs.DirEntry, err error) error {
//			if err != nil {
//				// 捕获权限错误并跳过该目录
//				if os.IsPermission(err) {
//					return nil
//				}
//				return err
//			}
//			// 检查目录是否需要跳过
//			if d.IsDir() {
//
//				skip := false
//				for _, skipDir := range utils.ConvertStringToList(skipDirs) {
//					if strings.Compare(path, skipDir) == 0 {
//						skip = true
//						break
//					}
//				}
//				if skip {
//					return filepath.SkipDir
//				}
//				// 过滤日期
//			} else {
//				// 修改此处，判断是否为符号链接，如果是，递归遍历链接目标
//				if d.Type()&fs.ModeSymlink != 0 {
//					linkTarget, err := os.Readlink(path)
//					if err != nil {
//						fmt.Printf("Error reading symlink: %v\n", err)
//						return nil
//					}
//					// 递归遍历链接目标
//					return WalkQuery(linkTarget, skipDirs, info)
//				}
//
//				flagQuery(&files, info, path, d)
//			}
//			return nil
//		},
//	)
//	if err != nil {
//		return nil, fmt.Errorf("Error walking directory: %v", err)
//	}
//
//	return files, nil
//}

// bool : false 表示 改文件不符合要求
func flagQuery(files *[]string, info *option.FlagInfo, path string, d fs.DirEntry) {
	if info.RootPath == "" {
		switch runtime.GOOS {
		case "windows":
			info.RootPath = "%UserProfile%"
			if info.SkipDirs == "" {
				info.SkipDirs = "C:\\Windows, C:\\Program Files, C:\\Program Files (x86), C:\\inetpub, C:\\Users\\Public"
			}
		case "linux", "darwin":
			info.RootPath = "~"
		}
	}

	if info.AfterDateStr != "" && !dateFilter(info, d) {
		return
	}
	if info.FileName != "" && !filenameFilter(info, d) {
		return
	}
	if info.Keyword != "" && !keywordFilter(info, d) {
		return
	}
	if !extFilter(info, d) {
		return
	}
	*files = append(*files, path)

}

func extFilter(info *option.FlagInfo, d fs.DirEntry) bool {
	// 获取文件扩展名
	ext := filepath.Ext(d.Name())
	var extensionsMap map[string]struct{}
	if info.Extension == "all" {
		extensionsMap = map[string]struct{}{
			".pdf": {}, ".docx": {}, ".doc": {}, ".xlsx": {}, ".xls": {}, ".csv": {},
			".pptx": {}, ".ppt": {}, ".zip": {}, ".rar": {}, ".7z": {}, ".tar": {}, ".gz": {}, ".tgz": {},
		}
	} else if info.Extension == "" {
		return true
	} else {
		extensionsMap = utils.StringToExtensionsMap(info.Extension)
	}
	// 检查文件扩展名是否匹配所需的扩展名
	if _, ok := extensionsMap[ext]; ok {
		return true
	}
	return false
}

// 如果有日期限制，检查修改时间是否在指定日期之后
func dateFilter(info *option.FlagInfo, d fs.DirEntry) bool {
	var afterDate time.Time
	afterDate, err := time.Parse("2006-01-02", info.AfterDateStr)
	if err != nil {
		fmt.Errorf("Failed to parse after date: %v", err)
		os.Exit(0)
	}
	fileInfo, _ := d.Info()
	if afterDate.IsZero() || fileInfo.ModTime().After(afterDate) {
		return true
	}
	return false
}

// 文件关键字匹配过滤
func keywordFilter(info *option.FlagInfo, d fs.DirEntry) bool {
	// 如果 info.FileName 包含逗号，则拆分成多个查询字符串
	queryStrings := strings.Split(info.FileName, ",")
	// 遍历所有查询字符串，检查是否包含在文件名中
	for _, queryString := range queryStrings {
		if strings.Contains(strings.ToLower(d.Name()), strings.ToLower(queryString)) {
			return true
		}
	}
	return false
}

// 文件名匹配过滤
func filenameFilter(info *option.FlagInfo, d fs.DirEntry) bool {
	// 如果 info.FileName 包含逗号，则拆分成多个查询字符串
	queryStrings := strings.Split(info.FileName, ",")
	// 遍历所有查询字符串，检查是否包含在文件名中
	for _, queryString := range queryStrings {
		if strings.Contains(strings.ToLower(d.Name()), strings.ToLower(queryString)) {
			return true
		}
	}
	return false
}
