package utils

import (
	"Fdoc/logx"
	"os"
)

// GetTotalSize returns the total size of the given files.
func GetTotalSize(files []string) int64 {
	totalSize := int64(0)
	for _, filePath := range files {
		fileInfo, err := os.Lstat(filePath)
		if err != nil {
			logx.Error("Unable to obtain file information %s: %v", filePath, err)
			continue
		}
		totalSize += fileInfo.Size()
	}
	return totalSize
}

func DeleteFile(path string) {
	if !IsFileExists(path) {
		return
	}
	if err := os.Remove(path); err != nil {
		logx.Error("delete failed: %s", path)
	}
}

func IsFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}
