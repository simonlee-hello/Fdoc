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

// FileFilter 文件过滤器结构体
type FileFilter struct {
	info *option.FlagInfo // 命令行参数信息
}

// NewFileFilter 创建一个新的文件过滤器
func NewFileFilter(info *option.FlagInfo) *FileFilter {
	return &FileFilter{info: info}
}

// Filter 过滤文件，根据多个条件进行过滤
func (ff *FileFilter) Filter(path string, d fs.DirEntry) bool {
	return ff.dateFilter(d) && ff.filenameFilter(d) && ff.keywordFilter(path) && ff.extFilter(d)
}

// extFilter 根据文件后缀进行过滤
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
			".bak": {}, ".bz2": {}, ".txt": {},
		}
	} else if ff.info.Extension == "documents" {
		extensionsMap = map[string]struct{}{
			".pdf": {}, ".docx": {}, ".doc": {}, ".xlsx": {}, ".xls": {}, ".csv": {},
			".pptx": {}, ".ppt": {},
		}
	} else if ff.info.Extension == "archives" {
		extensionsMap = map[string]struct{}{
			".zip": {}, ".rar": {}, ".7z": {}, ".tar": {}, ".gz": {}, ".tgz": {}, ".bak": {}, ".bz2": {},
		}
	} else if ff.info.Extension == "images" {
		extensionsMap = map[string]struct{}{
			".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".bmp": {},
		}
	} else if ff.info.Extension == "videos" {
		extensionsMap = map[string]struct{}{
			".mp4": {}, ".mkv": {}, ".avi": {}, ".mov": {},
		}
	}
	_, ok := extensionsMap[ext]
	return ok
}

// dateFilter 根据文件修改时间进行过滤
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
		gologger.Error().Msgf("open file error：%v", err)
		return false
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	keyword := strings.ToLower(ff.info.Keyword)
	for {
		line, err := reader.ReadString('\n')
		if strings.Contains(strings.ToLower(line), keyword) {
			return true
		}
		if err != nil {
			if err.Error() != "EOF" {
				gologger.Error().Msgf("error when reading file：%v", err)
			}
			break
		}
	}
	return false
}
