package apis

import (
	"uploader/apis/methods"
)

var (
	transferConfig methods.TransferConfig
	DebugMode      bool
	MuteMode       bool
	QuietMode      bool
	Output         string
	// SizeHint optionally returns alternative backends when size exceeds limit.
	SizeHint func(size int64) string
)

func TransferConfig() *methods.TransferConfig {
	return &transferConfig
}

func init() {
	transferConfig.DebugMode = &DebugMode
}
