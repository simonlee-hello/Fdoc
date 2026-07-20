package pkg

import (
	"Fdoc/logx"
	"Fdoc/option"
	"Fdoc/utils"
	"bufio"
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileFilter applies date/name/keyword/extension filters.
type FileFilter struct {
	info       *option.FlagInfo
	extensions map[string]struct{}
	keywords   []string
	filenames  []string
	afterDate  time.Time
	hasDate    bool
}

// NewFileFilter builds a FileFilter with cached filter criteria.
func NewFileFilter(info *option.FlagInfo) *FileFilter {
	ff := &FileFilter{info: info}
	ff.extensions = buildExtensionMap(info.Extension)
	if info.FileName != "" {
		ff.filenames = utils.ConvertStringToList(info.FileName)
	}
	if info.Keyword != "" {
		ff.keywords = utils.ConvertStringToList(info.Keyword)
	}
	if info.AfterDateStr != "" {
		afterDate, err := time.ParseInLocation("2006-01-02", info.AfterDateStr, time.Local)
		if err != nil {
			logx.Error("Failed to parse after date %q: %v", info.AfterDateStr, err)
		} else {
			ff.afterDate = afterDate
			ff.hasDate = true
		}
	}
	return ff
}

func buildExtensionMap(extension string) map[string]struct{} {
	extension = strings.ToLower(strings.TrimSpace(extension))
	if extension == "" || extension == "any" || extension == "*" {
		return nil
	}
	switch extension {
	case "all":
		// Common docs + archives + txt (NOT every file on disk).
		return map[string]struct{}{
			".pdf": {}, ".docx": {}, ".doc": {}, ".xlsx": {}, ".xls": {}, ".csv": {},
			".pptx": {}, ".ppt": {}, ".zip": {}, ".rar": {}, ".7z": {}, ".tar": {}, ".gz": {}, ".tgz": {},
			".bak": {}, ".bz2": {}, ".txt": {},
		}
	case "documents":
		return map[string]struct{}{
			".pdf": {}, ".docx": {}, ".doc": {}, ".xlsx": {}, ".xls": {}, ".csv": {},
			".pptx": {}, ".ppt": {},
		}
	case "archives", "packages":
		return map[string]struct{}{
			".zip": {}, ".rar": {}, ".7z": {}, ".tar": {}, ".gz": {}, ".tgz": {}, ".bak": {}, ".bz2": {},
		}
	case "images":
		return map[string]struct{}{
			".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".bmp": {},
		}
	case "videos":
		return map[string]struct{}{
			".mp4": {}, ".mkv": {}, ".avi": {}, ".mov": {},
		}
	default:
		return utils.StringToMap(extension)
	}
}

// Filter returns true when the file matches all active criteria (AND).
// Cheap checks run first; content keyword scan runs last.
func (ff *FileFilter) Filter(path string, d fs.DirEntry) bool {
	return ff.extFilter(d) && ff.dateFilter(d) && ff.filenameFilter(d) && ff.keywordFilter(path)
}

func (ff *FileFilter) extFilter(d fs.DirEntry) bool {
	if ff.extensions == nil {
		return true
	}
	ext := strings.ToLower(filepath.Ext(d.Name()))
	_, ok := ff.extensions[ext]
	return ok
}

func (ff *FileFilter) dateFilter(d fs.DirEntry) bool {
	if !ff.hasDate {
		return true
	}
	fileInfo, err := d.Info()
	if err != nil {
		return false
	}
	// Include files modified on the given day (local midnight) or later.
	return !fileInfo.ModTime().Before(ff.afterDate)
}

func (ff *FileFilter) filenameFilter(d fs.DirEntry) bool {
	if len(ff.filenames) == 0 {
		return true
	}
	name := strings.ToLower(d.Name())
	for _, query := range ff.filenames {
		if query == "" {
			continue
		}
		if strings.Contains(name, strings.ToLower(query)) {
			return true
		}
	}
	return false
}

func (ff *FileFilter) keywordFilter(path string) bool {
	if len(ff.keywords) == 0 {
		return true
	}

	file, err := os.Open(path)
	if err != nil {
		// Inaccessible / unreadable files simply do not match.
		logx.Debug("keyword open skip: %s (%v)", path, err)
		return false
	}
	defer file.Close()

	// Skip obvious binary files (NUL in the first 512 bytes).
	header := make([]byte, 512)
	n, _ := io.ReadFull(file, header)
	if n > 0 && bytes.IndexByte(header[:n], 0) >= 0 {
		return false
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		logx.Debug("keyword seek skip: %s (%v)", path, err)
		return false
	}

	const maxScanBytes = 8 << 20 // 8MB
	reader := bufio.NewReader(io.LimitReader(file, maxScanBytes))
	lowerKeywords := make([]string, 0, len(ff.keywords))
	for _, kw := range ff.keywords {
		kw = strings.TrimSpace(kw)
		if kw != "" {
			lowerKeywords = append(lowerKeywords, strings.ToLower(kw))
		}
	}
	if len(lowerKeywords) == 0 {
		return true
	}

	for {
		line, err := reader.ReadString('\n')
		lowerLine := strings.ToLower(line)
		for _, kw := range lowerKeywords {
			if strings.Contains(lowerLine, kw) {
				return true
			}
		}
		if err != nil {
			if err != io.EOF {
				logx.Debug("keyword read skip: %s (%v)", path, err)
			}
			break
		}
	}
	return false
}
