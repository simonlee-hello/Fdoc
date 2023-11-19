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

// bool : false 表示 该文件不符合要求
func fileFilter(info *option.FlagInfo, path string, d fs.DirEntry) bool {

	if info.AfterDateStr != "" && !dateFilter(info, d) {
		return false
	}
	if info.FileName != "" && !filenameFilter(info, d) {
		return false
	}
	if info.Keyword != "" && !keywordFilter(info, path) {
		return false
	}
	if !extFilter(info, d) {
		return false
	}
	return true

}

// 根据后缀进行过滤
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
		extensionsMap = utils.StringToMap(info.Extension)
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
func keywordFilter(info *option.FlagInfo, path string) bool {
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
		if strings.Contains(strings.ToLower(line), strings.ToLower(info.Keyword)) {
			return true
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("读取文件时发生错误：", err)
		return false
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
