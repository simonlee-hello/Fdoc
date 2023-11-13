package pkg

import "fmt"

// LinkError 是一个自定义的错误类型，表示找到了链接文件
type LinkError struct {
	Path string
}

// 实现 error 接口的 Error 方法
func (e *LinkError) Error() string {
	return fmt.Sprintf("link file: %s", e.Path)
}

// OverSizeError 是一个自定义的错误类型，表示总大小超过了预设值
type OverSizeError struct {
	MaxSize string
}

// 实现 error 接口的 Error 方法
func (e *OverSizeError) Error() string {
	return fmt.Sprintf("the total size is greater than: %s", e.MaxSize)
}
