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

func TestAssignmentVariants_IncludesJSONQuotes(t *testing.T) {
	vars := assignmentVariants("password")
	need := []string{`password:`, `password=`, `password :`, `password =`, `"password":`, `"password" :`, `'password':`}
	for _, n := range need {
		found := false
		for _, v := range vars {
			if v == n {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("assignmentVariants missing %q in %v", n, vars)
		}
	}
}

func TestExpandKeywordTokens_SecretsPreset(t *testing.T) {
	got := expandKeywordTokens("secrets")
	if len(got) < 20 {
		t.Fatalf("secrets preset should expand to many variants, got %d", len(got))
	}
	hasJSON := false
	hasBare := false
	for _, kw := range got {
		if kw == `"password":` {
			hasJSON = true
		}
		if kw == `password=` {
			hasBare = true
		}
	}
	if !hasJSON || !hasBare {
		t.Fatalf("secrets expand missing JSON/bare forms: json=%v bare=%v", hasJSON, hasBare)
	}

	creds := expandKeywordTokens("CREDS")
	if len(creds) != len(got) {
		t.Fatalf("creds alias should match secrets size: %d vs %d", len(creds), len(got))
	}
}

func TestExpandKeywordTokens_MixPresetAndLiteral(t *testing.T) {
	got := expandKeywordTokens("secrets,corp_sso=")
	found := false
	for _, kw := range got {
		if kw == "corp_sso=" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected literal corp_sso= kept, got %v", got)
	}
}

func TestKeywordFilter_SecretsMatchesJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{"password":"xxx","user":"a"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ff := NewFileFilter(&option.FlagInfo{Keyword: "secrets"})
	if !ff.keywordFilter(path) {
		t.Fatal(`secrets preset should match JSON "password":"xxx"`)
	}
}

func TestKeywordFilter_SecretsMatchesBareAssign(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.ini")
	if err := os.WriteFile(path, []byte("password = secret123\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ff := NewFileFilter(&option.FlagInfo{Keyword: "secrets"})
	if !ff.keywordFilter(path) {
		t.Fatal("secrets preset should match password = value")
	}
}

func TestKeywordFilter_SecretsSkipsLooseWord(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "policy.txt")
	if err := os.WriteFile(path, []byte("See Password-Policy for details\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ff := NewFileFilter(&option.FlagInfo{Keyword: "secrets"})
	if ff.keywordFilter(path) {
		t.Fatal("secrets should not match bare Password-Policy without assign form")
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
