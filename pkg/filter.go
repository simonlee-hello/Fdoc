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
		ff.keywords = expandKeywordTokens(info.Keyword)
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

// secretPresetKeys are base names expanded by the secrets/creds keyword preset.
var secretPresetKeys = []string{
	"password", "passwd", "pwd", "token",
	"api_key", "access_key", "secret_key", "client_secret", "private_key", "aws_secret",
}

var secretPresetKeysCN = []string{"密码", "口令"}

// expandKeywordTokens splits -keyword/-k by comma; known presets expand to assign/JSON variants.
func expandKeywordTokens(raw string) []string {
	parts := utils.ConvertStringToList(raw)
	out := make([]string, 0, len(parts)*16)
	seen := make(map[string]struct{}, len(parts)*16)
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, part := range parts {
		switch strings.ToLower(part) {
		case "secrets", "creds":
			for _, k := range secretPresetKeys {
				for _, v := range assignmentVariants(k) {
					add(v)
				}
			}
			for _, k := range secretPresetKeysCN {
				for _, v := range assignmentVariantsCN(k) {
					add(v)
				}
			}
		default:
			add(part)
		}
	}
	return out
}

// assignmentVariants builds bare / double-quoted / single-quoted assign forms for an ASCII key.
func assignmentVariants(key string) []string {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	seps := []string{":", "=", " :", " ="}
	out := make([]string, 0, len(seps)*3)
	for _, sep := range seps {
		out = append(out, key+sep, `"`+key+`"`+sep, `'`+key+`'`+sep)
	}
	return out
}

// assignmentVariantsCN adds half-width and full-width separators, plus JSON double quotes.
func assignmentVariantsCN(key string) []string {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	seps := []string{":", "=", " :", " =", "：", "＝", " ：", " ＝"}
	out := make([]string, 0, len(seps)*2)
	for _, sep := range seps {
		out = append(out, key+sep, `"`+key+`"`+sep)
	}
	return out
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
