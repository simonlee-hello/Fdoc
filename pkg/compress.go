package pkg

import (
	"Fdoc/option"
	"Fdoc/pkg/compress"
	"Fdoc/utils"
	"github.com/projectdiscovery/gologger"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func WalkAndCompress(info *option.FlagInfo) {
	var tarGzWriter *compress.TarGzWriter

	if !info.Size {
		var err error
		tarGzWriter, err = compress.NewTarGzWriter(info.OutputPath)
		if err != nil {
			gologger.Error().Str("err", err.Error()).Msg("NewTarGzWriter error")
			return
		}
		defer tarGzWriter.Close()
	}

	// 用来计算文件总大小
	var totalSizeBytes int64

	walker := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// 捕获权限错误并跳过该目录
			if os.IsPermission(err) {
				gologger.Warning().Msgf("permission error: %v", err)
				return nil
			}
			gologger.Warning().Msgf("Error while traversing the directory：%v", err)
			return nil
		}

		// 检查目录是否需要跳过
		if d.IsDir() {
			skip := false
			if info.SkipDirs != "" {
				for _, skipDir := range utils.ConvertStringToList(info.SkipDirs) {
					skipDir = filepath.Join(info.RootPath, skipDir)
					if strings.Compare(path, skipDir) == 0 {
						skip = true
						break
					}
				}
			}
			if skip {
				return filepath.SkipDir
			}

			// 过滤日期
		} else {
			// 修改此处，判断是否为符号链接，如果是，递归遍历链接目标
			if d.Type()&fs.ModeSymlink != 0 {
				//return &LinkError{Path: path}
				return nil
				// TODO 先跳过 后面写逻辑
				//linkTarget, err := os.Readlink(path)
				//if err != nil {
				//	gologger.Error().Msgf("获取符合链接错误,err:%v", err)
				//	return err
				//}
				//// 如果 linkTarget 是相对路径，则转换为绝对路径
				//if !filepath.IsAbs(linkTarget) {
				//	linkTarget = filepath.Join(filepath.Dir(path), linkTarget)
				//}
				//gologger.Debug().Str("absLinkTarget", linkTarget)
				//if utils.IsDir(linkTarget) {
				//
				//	//return &LinkError{Path: path}
				//	//return filepath.WalkDir(linkTarget, walkFunc)
				//}
				//if utils.IsFileExists(linkTarget) {
				//	FileToTarGz(linkTarget, info.RootPath, tarWriter)
				//}

			}

			// 不是符号链接，正常处理
			if f := NewFileFilter(info); f.Filter(path, d) {
				gologger.Verbose().Msgf("files: %v", path)
				// 计算文件大小
				fileInfo, err := os.Lstat(path)
				if err != nil {
					gologger.Error().Msgf("Unable to obtain file information %s: %v\n", path, err)
				}
				totalSizeBytes = totalSizeBytes + fileInfo.Size()
				if totalSizeBytes > utils.SizeToBytes(info.MaxSize) {
					return &OverSizeError{info.MaxSize}
				}
				// 打包
				if utils.IsFileExists(path) && !info.Size {
					compress.FileToTarGz(path, info.RootPath, tarGzWriter.TarWriter)
				}

			}
		}

		return nil
	}
	// 遍历目录并执行 walker 函数
	err := filepath.WalkDir(info.RootPath, walker)
	if err != nil {
		gologger.Error().Str("err", err.Error()).Msg("Error walking directory")
		utils.DeleteFile(info.OutputPath)
		return
	}

	// 打印文件总大小
	if info.Size {
		totalSize := utils.BytesToSize(totalSizeBytes)
		gologger.Info().Msgf("totalSize:%v", totalSize)
		gologger.Info().Msg("exiting..")
		return
	}
	// 计算打包好的tar.gz文件大小
	tarGzSize := utils.BytesToSize(utils.GetTotalSize([]string{info.OutputPath}))
	if tarGzSize == "0.00 Bytes" {
		gologger.Info().Msg("file not found")
		utils.DeleteFile(info.OutputPath)
	} else {
		gologger.Info().Str("path", info.OutputPath).Str("size", tarGzSize).Msg("SUCCESS!")
	}

}
