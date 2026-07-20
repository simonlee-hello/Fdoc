package utils

import (
	"archive/zip"
	"compress/flate"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ZipDir writes a Deflate-compressed zip of rootDir into destZip.
// Paths inside the archive are relative to rootDir.
func ZipDir(rootDir, destZip string) error {
	info, err := os.Stat(rootDir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", rootDir)
	}
	if err := os.MkdirAll(filepath.Dir(destZip), 0755); err != nil {
		return err
	}
	out, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	zw.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, flate.BestCompression)
	})
	defer zw.Close()

	rootDir = filepath.Clean(rootDir)
	return filepath.Walk(rootDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == rootDir {
			return nil
		}
		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if fi.IsDir() {
			_, err := zw.CreateHeader(&zip.FileHeader{
				Name:   rel + "/",
				Method: zip.Store,
			})
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		hdr := &zip.FileHeader{
			Name:     rel,
			Method:   zip.Deflate,
			Modified: fi.ModTime(),
		}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(w, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

// ZipDirTemp packs rootDir into a temp *.zip. Caller must os.Remove the path.
func ZipDirTemp(rootDir string) (string, error) {
	base := filepath.Base(filepath.Clean(rootDir))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "archive"
	}
	f, err := os.CreateTemp("", base+"-*.zip")
	if err != nil {
		return "", err
	}
	path := f.Name()
	_ = f.Close()
	_ = os.Remove(path)
	if err := ZipDir(rootDir, path); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}
