package compress

import (
	"Fdoc/utils"
	"archive/tar"
	"compress/gzip"
	"github.com/projectdiscovery/gologger"
	"io"
	"os"
	"path/filepath"
)

// TarGzWriter 封装了 tar.Writer 和其相关的资源
type TarGzWriter struct {
	TarWriter *tar.Writer
	gzWriter  *gzip.Writer
	file      *os.File
}

// NewTarGzWriter 创建一个新的 TarGzWriter
func NewTarGzWriter(outputPath string) (*TarGzWriter, error) {
	// 创建一个输出tar+gzip归档文件
	file, err := os.Create(outputPath)
	if err != nil {
		gologger.Error().Msgf("output file create failed: %v", err)
		return nil, err
	}

	// 创建一个gzip写入器
	gzWriter, err := gzip.NewWriterLevel(file, gzip.BestSpeed)
	if err != nil {
		gologger.Error().Msgf("gzip writer creation failed: %v", err)
		_ = file.Close() // 关闭文件
		return nil, err
	}

	// 创建一个tar写入器
	tarWriter := tar.NewWriter(gzWriter)

	return &TarGzWriter{
		TarWriter: tarWriter,
		gzWriter:  gzWriter,
		file:      file,
	}, nil
}

// Close 关闭所有相关的资源
func (tw *TarGzWriter) Close() error {
	var errors []error

	if tw.TarWriter != nil {
		if err := tw.TarWriter.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	if tw.gzWriter != nil {
		if err := tw.gzWriter.Close(); err != nil {
			errors = append(errors, err)
		}
	}
	if tw.file != nil {
		if err := tw.file.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		// 在这里处理所有的关闭错误，例如记录日志或返回错误
		return errors[0] // 这里简单返回第一个错误，你也可以根据需求处理所有错误
	}

	return nil
}

func FileToTarGz(filePath string, rootDir string, tarWriter *tar.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		gologger.Warning().Msgf("Unable to open the file %s: %v\n", filePath, err)
		return
	}
	defer file.Close()

	// 获取文件信息
	info, err := file.Stat()
	if err != nil {
		gologger.Warning().Msgf("Failed to obtain file information: %v\n", err)
		return
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
