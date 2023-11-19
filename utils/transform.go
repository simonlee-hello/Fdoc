package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// 将输入的字符串解析为数字和单位，然后根据单位将其转换为字节数
func SizeToBytes(maxsize string) int64 {
	// 去掉空格并转为大写
	maxsize = strings.ToUpper(strings.ReplaceAll(maxsize, " ", ""))

	// 分离数字和单位
	numStr := maxsize
	unit := ""
	for i, char := range maxsize {
		if char < '0' || char > '9' {
			numStr = maxsize[:i]
			unit = maxsize[i:]
			break
		}
	}

	// 解析数字部分
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	// 根据单位转换为字节
	switch unit {
	case "KB":
		return int64(num * 1024)
	case "MB":
		return int64(num * 1024 * 1024)
	case "GB":
		return int64(num * 1024 * 1024 * 1024)
	default:
		// 默认情况下，认为是字节
		return int64(num)
	}
}

func BytesToSize(bytes int64) string {
	// 根据字节数大小，确定使用的单位
	unit := "Bytes"
	value := float64(bytes)

	if bytes >= 1024*1024*1024 {
		unit = "GB"
		value = float64(bytes) / (1024 * 1024 * 1024)
	} else if bytes >= 1024*1024 {
		unit = "MB"
		value = float64(bytes) / (1024 * 1024)
	} else if bytes >= 1024 {
		unit = "KB"
		value = float64(bytes) / 1024
	}
	return fmt.Sprintf("%.2f %s", value, unit)

}

func ConvertStringToList(input string) []string {
	// 以逗号分隔字符串
	parts := strings.Split(input, ",")

	// 初始化结果切片
	result := make([]string, 0, len(parts))

	// 去掉每个部分的前后空白，并添加到结果切片
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		result = append(result, trimmedPart)
	}

	return result
}

// 将字符串转为map，例如："docx,pdf,xlsx" -> {".pdf": {}, ".docx": {}, ".doc": {}, ".xlsx": {}}
func StringToMap(supportedExtensions string) map[string]struct{} {
	m := make(map[string]struct{})

	// 将以逗号分隔的字符串分割成切片
	list := strings.Split(supportedExtensions, ",")

	// 遍历切片并将每个扩展名添加到映射中
	for _, ext := range list {
		// 去除空格
		ext = strings.TrimSpace(ext)
		if ext != "" {
			// 将扩展名转换为小写并添加到映射中
			m["."+strings.ToLower(ext)] = struct{}{}
		}
	}

	return m
}

func TransformSlash(input string) string {
	return strings.Replace(input, `\`, `/`, -1)
}
