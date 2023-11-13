package pkg

import (
	"Fdoc/option"
	"Fdoc/utils"
	"archive/tar"
	"compress/gzip"
	"errors"
	"github.com/projectdiscovery/gologger"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func QueryAndCompress(info *option.FlagInfo) {
	var tarWriter *tar.Writer

	if !info.Size {
		// 创建一个输出tar+gzip归档文件
		tarGzFile, err := os.Create(info.OutputPath)
		if err != nil {
			gologger.Error().Msgf("output file create failed")
			return
		}
		defer tarGzFile.Close()

		// 创建一个gzip写入器
		//gzWriter := gzip.NewWriter(tarGzFile)
		gzWriter, _ := gzip.NewWriterLevel(tarGzFile, gzip.BestSpeed)

		defer gzWriter.Close()

		// 创建一个tar写入器
		tarWriter = tar.NewWriter(gzWriter)
		defer tarWriter.Close()
	}

	// 用来计算文件总大小
	var totalSizeBytes int64
	// 定义 WalkDir 函数
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
				return nil
				// TODO 先跳过 后面写逻辑
				return &LinkError{Path: path}

			}

			// 不是符号链接，正常处理
			if fileFilter(info, path, d) {
				gologger.Verbose().Msgf("files: %v", path)
				// 计算文件大小
				fileInfo, err := os.Lstat(path)
				if err != nil {
					gologger.Error().Msgf("Unable to obtain file information %s: %v\n", path, err)
				}
				totalSizeBytes = totalSizeBytes + fileInfo.Size()
				if totalSizeBytes > utils.SizeToBytes(info.MaxSize) {
					return &OverSizeError{MaxSize: info.MaxSize} // 返回标志表示超过了预设值
				}
				// 打包
				FileToTarGz(path, info.RootPath, tarWriter)

			}
		}

		return nil
	}
	// 遍历目录并执行 walker 函数
	err := filepath.WalkDir(info.RootPath, walker)
	if err != nil {
		var linkErr *LinkError
		var sizeErr *OverSizeError

		// 检查错误是否为 LinkError 类型
		if errors.As(err, &linkErr) {
			// 处理链接文件的逻辑
			gologger.Warning().Msgf("处理链接文件:", linkErr.Path)

			//filepath.WalkDir(linkErr.Path, walker)
		} else if errors.As(err, &sizeErr) {
			// 处理总大小超过预设值的逻辑
			gologger.Error().Msgf("处理大小超过预设值的错误:", sizeErr.MaxSize)
			utils.DeleteFile(info.OutputPath)
			return
		} else {
			// 处理其他错误
			gologger.Error().Msgf("Error walking directory:", err)
			utils.DeleteFile(info.OutputPath)
			return
		}
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

//func QueryAndCompress(info *option.FlagInfo) {
//
//	var files []string
//	var err error
//
//	// 遍历并查询出所有files path
//	rootPath := info.RootPath
//	skipDIrs := info.SkipDirs
//	files = WalkQuery(rootPath, skipDIrs, info)
//
//	//fmt.Println(files)
//	gologger.Verbose().Msgf("files: %v", files)
//
//	if err != nil {
//		gologger.Error().Str("err", err.Error()).Msg("WalkQuery function error")
//		return
//	}
//	if files == nil {
//		gologger.Warning().Msg("file not found")
//		gologger.Info().Msg("exiting..")
//		return
//	}
//
//	// 如果命令参数包含--size，则计算所有符合条件的文件总大小并返回
//	if info.Size {
//		// Execute the GetTotalSize function
//		totalSize := utils.BytesToSize(utils.GetTotalSize(files))
//		gologger.Info().Msgf("totalSize:%v", totalSize)
//		gologger.Info().Msg("exiting..")
//		return
//	}
//
//	totalSize, limit := utils.GetTotalSizeAndCheckLimit(files, utils.SizeToBytes(info.MaxSize))
//	if limit {
//		//fmt.Printf("totalSize more than %s, exit!\n", info.MaxSize)
//		gologger.Error().Msgf("totalSize more than %s, exit!\n", info.MaxSize)
//		return
//	}
//	gologger.Info().Msgf("totalSize:%v\n", utils.BytesToSize(totalSize))
//
//	//utils.FilesToZip(Info.RootPath, Info.OutputPath, files)
//	FilesToTarGz(info.RootPath, info.OutputPath, files)
//
//}
//
//// 将多个文件（files []string）打包到一个tar+gzip归档中
//func FilesToTarGz(rootDir string, tarGzPath string, files []string) {
//
//	// 创建一个输出tar+gzip归档文件
//	tarGzFile, err := os.Create(tarGzPath)
//	if err != nil {
//		gologger.Error().Msgf("output file create failed")
//		return
//	}
//	defer tarGzFile.Close()
//
//	// 创建一个gzip写入器
//	//gzWriter := gzip.NewWriter(tarGzFile)
//	gzWriter, _ := gzip.NewWriterLevel(tarGzFile, gzip.BestSpeed)
//
//	defer gzWriter.Close()
//
//	// 创建一个tar写入器
//	tarWriter := tar.NewWriter(gzWriter)
//	defer tarWriter.Close()
//
//	// 遍历文件列表并将它们添加到tar归档中
//	for _, filePath := range files {
//		file, err := os.Open(filePath)
//		if err != nil {
//			gologger.Warning().Msgf("Unable to open the file %s: %v\n", filePath, err)
//			continue
//		}
//		defer file.Close()
//
//		// 获取文件信息
//		info, err := file.Stat()
//		if err != nil {
//			gologger.Warning().Msgf("Failed to obtain file information: %v\n", err)
//			continue
//		}
//
//		// 创建tar头
//		header := new(tar.Header)
//		header.Name, _ = filepath.Rel(rootDir, filePath)
//		header.Name = utils.TransformSlash(header.Name)
//		header.Size = info.Size()
//		header.Mode = int64(info.Mode())
//		header.ModTime = info.ModTime()
//
//		// 将头部写入tar归档
//		if err := tarWriter.WriteHeader(header); err != nil {
//			gologger.Warning().Msgf("Failed to write tar header: %v\n", err)
//			continue
//		}
//
//		// 将文件内容拷贝到tar归档中
//		_, err = io.Copy(tarWriter, file)
//		if err != nil {
//			gologger.Warning().Msgf("Unable to copy %s to tar archive: %v\n", header.Name, err)
//			continue
//		}
//	}
//	tarGzSize := utils.BytesToSize(utils.GetTotalSize([]string{tarGzPath}))
//
//	gologger.Info().Str("path", tarGzPath).Str("size", tarGzSize).Msg("SUCCESS!")
//}

func FileToTarGz(filePath string, rootDir string, tarWriter *tar.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		gologger.Warning().Msgf("Unable to open the file %s: %v\n", filePath, err)
	}
	defer file.Close()

	// 获取文件信息
	info, err := file.Stat()
	if err != nil {
		gologger.Warning().Msgf("Failed to obtain file information: %v\n", err)
	}

	// 创建tar头
	header := new(tar.Header)
	header.Name, _ = filepath.Rel(rootDir, filePath)
	header.Name = utils.TransformSlash(header.Name)
	header.Size = info.Size()
	header.Mode = int64(info.Mode())
	header.ModTime = info.ModTime()

	// 将头部写入tar归档
	if err := tarWriter.WriteHeader(header); err != nil {
		gologger.Warning().Msgf("Failed to write tar header: %v\n", err)
	}

	// 将文件内容拷贝到tar归档中
	_, err = io.Copy(tarWriter, file)
	if err != nil {
		gologger.Warning().Msgf("Unable to copy %s to tar archive: %v\n", header.Name, err)
	}
}
