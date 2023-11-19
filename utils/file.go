package utils

import (
	"github.com/projectdiscovery/gologger"
	"os"
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
