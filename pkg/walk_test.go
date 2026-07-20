package pkg

import (
	"Fdoc/option"
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFilter_ExtBeforeKeyword(t *testing.T) {
	dir := t.TempDir()
	// Wrong extension for documents preset, but contains keyword.
	path := filepath.Join(dir, "secret.bin")
	if err := os.WriteFile(path, []byte("token:should-not-scan-if-ext-fails\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ff := NewFileFilter(&option.FlagInfo{Extension: "documents", Keyword: "token:"})
	if ff.Filter(path, fakeDirEntry{name: "secret.bin"}) {
		t.Fatal("documents preset must reject .bin before keyword match")
	}
}

func TestKeywordFilter_SkipsBinary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.bin")
	data := append([]byte("token:abc"), 0x00, 0x01, 0x02)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	ff := NewFileFilter(&option.FlagInfo{Keyword: "token:"})
	if ff.keywordFilter(path) {
		t.Fatal("binary file with NUL should be skipped by keyword filter")
	}
}

func TestDateFilter_IncludesSameDay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "same.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	day := time.Now().Local().Format("2006-01-02")
	// Set mtime to local midnight of today
	midnight, _ := time.ParseInLocation("2006-01-02", day, time.Local)
	if err := os.Chtimes(path, midnight, midnight); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	ff := NewFileFilter(&option.FlagInfo{AfterDateStr: day})
	entry := &statDirEntry{name: "same.txt", fi: fi}
	if !ff.dateFilter(entry) {
		t.Fatal("file modified at local midnight of -t day should be included")
	}
}

type statDirEntry struct {
	name string
	fi   os.FileInfo
}

func (s *statDirEntry) Name() string               { return s.name }
func (s *statDirEntry) IsDir() bool                { return false }
func (s *statDirEntry) Type() os.FileMode          { return 0 }
func (s *statDirEntry) Info() (os.FileInfo, error) { return s.fi, nil }

func TestExtFilter_AnyMeansNoExtFilter(t *testing.T) {
	ff := NewFileFilter(&option.FlagInfo{Extension: "any"})
	if !ff.extFilter(fakeDirEntry{name: "a.xyz"}) {
		t.Fatal("any should allow all extensions")
	}
}

func TestWalk_MaxBestEffortKeepsArchive(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out.tar.gz")
	// Three small text docs; max allows roughly two.
	for _, name := range []string{"a.pdf", "b.pdf", "c.pdf"} {
		// 600 bytes each; max 1000 bytes => first file ok, second may fit or truncate
		if err := os.WriteFile(filepath.Join(root, name), []byte(strings.Repeat("x", 600)), 0644); err != nil {
			t.Fatal(err)
		}
	}
	info := &option.FlagInfo{
		RootPath:    root,
		OutputPath:  out,
		Extension:   "documents",
		MaxSize:     "1000",
		MaxFileSize: "0",
	}
	result := WalkAndCompress(info)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if !result.Truncated {
		t.Fatal("expected truncated=true")
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("archive should be kept: %v", err)
	}
	n := countTarMembers(t, out)
	if n < 1 {
		t.Fatalf("expected at least 1 packed file, got %d", n)
	}
	if n > 2 {
		t.Fatalf("expected at most 2 packed files under 1000B budget, got %d", n)
	}
}

func TestWalk_SkipsOutputArchiveInSameDir(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "out.tar.gz")
	if err := os.WriteFile(filepath.Join(root, "a.pdf"), []byte("%PDF-1.4 hello"), 0644); err != nil {
		t.Fatal(err)
	}
	// Also a .gz that would match -e all / gz; output must still be excluded.
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("plain"), 0644); err != nil {
		t.Fatal(err)
	}

	info := &option.FlagInfo{
		RootPath:    root,
		OutputPath:  out,
		Extension:   "any",
		MaxSize:     "1GB",
		MaxFileSize: "0",
	}
	result := WalkAndCompress(info)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if result.Truncated {
		t.Fatal("unexpected truncate")
	}
	names := tarMemberNames(t, out)
	for _, n := range names {
		if n == "out.tar.gz" || strings.HasSuffix(n, "/out.tar.gz") {
			t.Fatalf("output archive packed into itself: %v", names)
		}
	}
	if len(names) < 2 {
		t.Fatalf("expected other files packed, got %v", names)
	}
}

func TestWalk_MaxFileSkipsLarge(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out.tar.gz")
	if err := os.WriteFile(filepath.Join(root, "small.pdf"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "big.pdf"), []byte(strings.Repeat("y", 2000)), 0644); err != nil {
		t.Fatal(err)
	}
	info := &option.FlagInfo{
		RootPath:    root,
		OutputPath:  out,
		Extension:   "documents",
		MaxSize:     "1GB",
		MaxFileSize: "1000",
	}
	result := WalkAndCompress(info)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if result.Truncated {
		t.Fatal("should not truncate")
	}
	names := tarMemberNames(t, out)
	if len(names) != 1 || names[0] != "small.pdf" {
		t.Fatalf("expected only small.pdf, got %v", names)
	}
}

func TestWalk_CloseBeforeSizeReport(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out.tar.gz")
	payload := []byte(strings.Repeat("z", 4096))
	if err := os.WriteFile(filepath.Join(root, "a.pdf"), payload, 0644); err != nil {
		t.Fatal(err)
	}
	info := &option.FlagInfo{
		RootPath:    root,
		OutputPath:  out,
		Extension:   "documents",
		MaxSize:     "1GB",
		MaxFileSize: "0",
	}
	result := WalkAndCompress(info)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	fi, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	// Unflushed gzip often shows ~10 bytes; a real closed archive is larger.
	if fi.Size() < 50 {
		t.Fatalf("archive too small after close: %d bytes (likely unread flush)", fi.Size())
	}
}

func TestWalk_TruncateWithZeroFiles_ExitTruncated(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out.tar.gz")
	// Single file larger than -max => truncate before packing anything.
	if err := os.WriteFile(filepath.Join(root, "big.pdf"), []byte(strings.Repeat("x", 500)), 0644); err != nil {
		t.Fatal(err)
	}
	info := &option.FlagInfo{
		RootPath:    root,
		OutputPath:  out,
		Extension:   "documents",
		MaxSize:     "100",
		MaxFileSize: "0",
	}
	result := WalkAndCompress(info)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if !result.Truncated {
		t.Fatal("expected Truncated=true when first file exceeds -max")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("expected empty truncated archive to be removed")
	}
}

func TestWalk_NoMatchDeletesEmptyArchive(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out.tar.gz")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	info := &option.FlagInfo{
		RootPath:    root,
		OutputPath:  out,
		Extension:   "documents", // .txt not included
		MaxSize:     "1GB",
		MaxFileSize: "0",
	}
	result := WalkAndCompress(info)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("expected empty archive removed, still exists")
	}
}

func countTarMembers(t *testing.T, path string) int {
	t.Helper()
	return len(tarMemberNames(t, path))
}

func tarMemberNames(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, hdr.Name)
	}
	return names
}
