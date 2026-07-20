package compress

import (
	"Fdoc/logx"
	"Fdoc/utils"
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

// TarGzWriter wraps tar/gzip writers and the output file.
type TarGzWriter struct {
	TarWriter *tar.Writer
	gzWriter  *gzip.Writer
	file      *os.File
	closed    bool
}

// NewTarGzWriter creates a streaming tar.gz writer.
func NewTarGzWriter(outputPath string) (*TarGzWriter, error) {
	file, err := os.Create(outputPath)
	if err != nil {
		logx.Error("output file create failed: %v", err)
		return nil, err
	}

	gzWriter, err := gzip.NewWriterLevel(file, gzip.BestSpeed)
	if err != nil {
		logx.Error("gzip writer creation failed: %v", err)
		_ = file.Close()
		return nil, err
	}

	return &TarGzWriter{
		TarWriter: tar.NewWriter(gzWriter),
		gzWriter:  gzWriter,
		file:      file,
	}, nil
}

// Close closes the tar, gzip, and file writers. Safe to call multiple times.
func (tw *TarGzWriter) Close() error {
	if tw == nil || tw.closed {
		return nil
	}
	tw.closed = true

	var firstErr error
	closeOne := func(fn func() error) {
		if fn == nil {
			return
		}
		if err := fn(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if tw.TarWriter != nil {
		closeOne(tw.TarWriter.Close)
	}
	if tw.gzWriter != nil {
		closeOne(tw.gzWriter.Close)
	}
	if tw.file != nil {
		closeOne(tw.file.Close)
	}
	return firstErr
}

// FileToTarGz appends a single file into the tar archive.
func FileToTarGz(filePath string, rootDir string, tarWriter *tar.Writer) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	rel, err := filepath.Rel(rootDir, filePath)
	if err != nil {
		rel = filepath.Base(filePath)
	}

	header := &tar.Header{
		Name:    utils.TransformSlash(rel),
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(tarWriter, file)
	return err
}
