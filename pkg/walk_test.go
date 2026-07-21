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
		MaxSet:      true,
		MaxFileSize: "0",
		MaxFileSet:  true,
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

func TestWalk_ImplicitMaxRefusesTruncate(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "out.tar.gz")
	for _, name := range []string{"a.pdf", "b.pdf", "c.pdf"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(strings.Repeat("x", 600)), 0644); err != nil {
			t.Fatal(err)
		}
	}
	info := &option.FlagInfo{
		RootPath:    root,
		OutputPath:  out,
		Extension:   "documents",
		MaxSize:     "1000",
		MaxSet:      false, // default soft cap: must not silently truncate
		MaxFileSize: "0",
		MaxFileSet:  true,
	}
	result := WalkAndCompress(info)
	if result.Err == nil {
		t.Fatal("expected error when implicit -max would truncate")
	}
	if !strings.Contains(result.Err.Error(), "-max") {
		t.Fatalf("error should mention -max: %v", result.Err)
	}
	if result.Truncated {
		t.Fatal("must not report Truncated when refusing implicit default")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("partial archive must be deleted")
	}
}

func TestWalk_ImplicitMaxFileRefusesSkip(t *testing.T) {
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
		MaxSet:      true,
		MaxFileSize: "1000",
		MaxFileSet:  false, // default soft cap: must not silently skip
	}
	result := WalkAndCompress(info)
	if result.Err == nil {
		t.Fatal("expected error when implicit -max-file would skip")
	}
	if !strings.Contains(result.Err.Error(), "-max-file") {
		t.Fatalf("error should mention -max-file: %v", result.Err)
	}
	if !strings.Contains(result.Err.Error(), "big.pdf") {
		t.Fatalf("error should name the file: %v", result.Err)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("partial archive must be deleted")
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
		MaxSet:      true,
		MaxFileSize: "0",
		MaxFileSet:  true,
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
		MaxSet:      true,
		MaxFileSize: "1000",
		MaxFileSet:  true,
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
		MaxSet:      true,
		MaxFileSize: "0",
		MaxFileSet:  true,
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
		MaxSet:      true,
		MaxFileSize: "0",
		MaxFileSet:  true,
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

func TestWalk_FollowsSymlinkToRegularFile(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "scan")
	if err := os.Mkdir(root, 0755); err != nil {
		t.Fatal(err)
	}
	// Target outside walk root so only the symlink is visited.
	target := filepath.Join(outer, "real.pdf")
	payload := []byte("%PDF-1.4 symlink-target-content")
	if err := os.WriteFile(target, payload, 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "alias.pdf")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	out := filepath.Join(t.TempDir(), "out.tgz")
	info := &option.FlagInfo{
		RootPath:    root,
		OutputPath:  out,
		Extension:   "pdf",
		MaxSize:     "1GB",
		MaxFileSize: "0",
	}
	result := WalkAndCompress(info)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if result.MatchedFiles != 1 {
		t.Fatalf("matched=%d want 1", result.MatchedFiles)
	}
	names := tarMemberNames(t, out)
	foundLink := false
	for _, n := range names {
		if strings.Contains(n, "alias.pdf") {
			foundLink = true
		}
	}
	if !foundLink {
		t.Fatalf("expected alias.pdf in archive, got %v", names)
	}
	got := tarMemberContent(t, out, "alias.pdf")
	if string(got) != string(payload) {
		t.Fatalf("packed content=%q want target payload", got)
	}
}

func TestWalk_SkipsSymlinkToDir(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "subdir")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "hidden.pdf"), []byte("%PDF-1.4 in-dir"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "todir")
	if err := os.Symlink(dir, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	out := filepath.Join(t.TempDir(), "out.tgz")
	info := &option.FlagInfo{
		RootPath:    root,
		OutputPath:  out,
		Extension:   "pdf",
		MaxSize:     "1GB",
		MaxFileSize: "0",
	}
	result := WalkAndCompress(info)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	// WalkDir does not enter symlink dirs; symlink-to-dir is not a regular file.
	for _, n := range tarMemberNames(t, out) {
		if strings.Contains(n, "hidden.pdf") && strings.Contains(n, "todir") {
			t.Fatalf("must not pack via symlink dir: %v", n)
		}
	}
	// Real path under subdir should still be packed.
	found := false
	for _, n := range tarMemberNames(t, out) {
		if strings.Contains(n, "hidden.pdf") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected subdir/hidden.pdf packed, got %v", tarMemberNames(t, out))
	}
}

func countTarMembers(t *testing.T, path string) int {
	t.Helper()
	return len(tarMemberNames(t, path))
}

func tarMemberContent(t *testing.T, archive, wantSuffix string) []byte {
	t.Helper()
	f, err := os.Open(archive)
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
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			t.Fatalf("member ending with %q not found", wantSuffix)
		}
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(hdr.Name, wantSuffix) {
			b, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			return b
		}
	}
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
