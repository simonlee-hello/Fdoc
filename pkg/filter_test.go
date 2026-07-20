package pkg

import (
	"Fdoc/option"
	"os"
	"path/filepath"
	"testing"
)

type fakeDirEntry struct {
	name string
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return false }
func (f fakeDirEntry) Type() os.FileMode          { return 0 }
func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, os.ErrNotExist }

func TestFilenameFilter_CommaSeparated(t *testing.T) {
	ff := NewFileFilter(&option.FlagInfo{FileName: "pass,secret"})
	if !ff.filenameFilter(fakeDirEntry{name: "my_password.txt"}) {
		t.Fatal("expected match on first keyword fragment")
	}
	if !ff.filenameFilter(fakeDirEntry{name: "top_secret.doc"}) {
		t.Fatal("expected match on second keyword fragment")
	}
	if ff.filenameFilter(fakeDirEntry{name: "readme.md"}) {
		t.Fatal("expected no match")
	}
}

func TestKeywordFilter_CommaSeparated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello token:abc world\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ff := NewFileFilter(&option.FlagInfo{Keyword: "password:,token:"})
	if !ff.keywordFilter(path) {
		t.Fatal("expected match on comma-separated keyword token:")
	}

	ffMiss := NewFileFilter(&option.FlagInfo{Keyword: "password:,secret:"})
	if ffMiss.keywordFilter(path) {
		t.Fatal("expected no match when none of the keywords present")
	}
}

func TestKeywordFilter_SingleKeyword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(path, []byte("api_key=123\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ff := NewFileFilter(&option.FlagInfo{Keyword: "api_key"})
	if !ff.keywordFilter(path) {
		t.Fatal("expected single keyword match")
	}
}

func TestExtFilter_DocumentsPreset(t *testing.T) {
	ff := NewFileFilter(&option.FlagInfo{Extension: "documents"})
	if !ff.extFilter(fakeDirEntry{name: "a.pdf"}) {
		t.Fatal("documents preset should match .pdf")
	}
	if ff.extFilter(fakeDirEntry{name: "a.zip"}) {
		t.Fatal("documents preset should not match .zip")
	}
}

func TestExtFilter_PackagesAlias(t *testing.T) {
	ff := NewFileFilter(&option.FlagInfo{Extension: "packages"})
	if !ff.extFilter(fakeDirEntry{name: "a.zip"}) {
		t.Fatal("packages alias should match archives")
	}
}

func TestExtFilter_PresetCaseInsensitive(t *testing.T) {
	ff := NewFileFilter(&option.FlagInfo{Extension: "Documents"})
	if !ff.extFilter(fakeDirEntry{name: "a.pdf"}) {
		t.Fatal("Documents preset should match like documents")
	}
	ffAll := NewFileFilter(&option.FlagInfo{Extension: "ALL"})
	if !ffAll.extFilter(fakeDirEntry{name: "a.txt"}) {
		t.Fatal("ALL preset should match like all")
	}
}
