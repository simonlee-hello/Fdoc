package pkg

import (
	"Fdoc/option"
	"Fdoc/utils"
	"bufio"
	"fmt"
	"github.com/projectdiscovery/gologger"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 文件过滤器
type FileFilter struct {
	info *option.FlagInfo
}

// NewFileFilter 创建一个新的文件过滤器
func NewFileFilter(info *option.FlagInfo) *FileFilter {
	return &FileFilter{info: info}
}

// 过滤文件
func (ff *FileFilter) Filter(path string, d fs.DirEntry) bool {
	return ff.dateFilter(d) && ff.filenameFilter(d) && ff.keywordFilter(path) && ff.extFilter(d)
}

// 根据后缀进行过滤
func (ff *FileFilter) extFilter(d fs.DirEntry) bool {
	if ff.info.Extension == "" {
		return true
	}

	ext := filepath.Ext(d.Name())
	extensionsMap := utils.StringToMap(ff.info.Extension)
	if ff.info.Extension == "all" {
		extensionsMap = map[string]struct{}{
			".pdf": {}, ".docx": {}, ".doc": {}, ".xlsx": {}, ".xls": {}, ".csv": {},
			".pptx": {}, ".ppt": {}, ".zip": {}, ".rar": {}, ".7z": {}, ".tar": {}, ".gz": {}, ".tgz": {},
		}
	}
	_, ok := extensionsMap[ext]
	return ok
}

// 如果有日期限制，检查修改时间是否在指定日期之后
func (ff *FileFilter) dateFilter(d fs.DirEntry) bool {
	if ff.info.AfterDateStr == "" {
		return true
	}

	afterDate, err := time.Parse("2006-01-02", ff.info.AfterDateStr)
	if err != nil {
		fmt.Errorf("Failed to parse after date: %v", err)
		os.Exit(0)
	}

	fileInfo, _ := d.Info()
	return afterDate.IsZero() || fileInfo.ModTime().After(afterDate)
}

// 文件名匹配过滤
func (ff *FileFilter) filenameFilter(d fs.DirEntry) bool {
	if ff.info.FileName == "" {
		return true
	}

	queryStrings := strings.Split(ff.info.FileName, ",")
	for _, queryString := range queryStrings {
		if strings.Contains(strings.ToLower(d.Name()), strings.ToLower(queryString)) {
			return true
		}
	}
	return false
}

// 文件内容关键字匹配过滤
func (ff *FileFilter) keywordFilter(path string) bool {
	if ff.info.Keyword == "" {
		return true
	}

	file, err := os.Open(path)
	if err != nil {
		gologger.Error().Msgf("无法打开文件：%v", err)
		return false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// 将文件内容转换为小写，然后进行比较
		if strings.Contains(strings.ToLower(line), strings.ToLower(ff.info.Keyword)) {
			return true
		}
	}

	if err := scanner.Err(); err != nil {
		gologger.Error().Msgf("读取文件时发生错误：%v", err)
		return false
	}

	return false
}
