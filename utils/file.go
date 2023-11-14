package utils

import (
	"github.com/projectdiscovery/gologger"
	"os"
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

//func IsLink(file string) bool {
//	info, err := os.Lstat(file)
//	if err != nil {
//		gologger.Error().Str("err", err.Error()).Msg("IsLink error")
//		return false
//	}
//
//	if runtime.GOOS == "windows" && filepath.Ext(file) == ".lnk" {
//		// 在 Windows 上，检查文件的属性
//		return true
//	} else if runtime.GOOS == "linux" {
//		// 在 Linux上，检查文件的模式和IsDir方法
//		return (info.Mode()&os.ModeSymlink) != 0 || info.IsDir()
//	} else {
//		return false
//	}
//}

func TransformSlash(input string) string {
	return strings.Replace(input, `\`, `/`, -1)
}

func DeleteFile(path string) {
	if IsFileExists(path) {
		err := os.Remove(path)
		if err != nil {
			gologger.Error().Str("path", path).Msg("delete failed")
			return
		}
	}

}

func IsFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}
