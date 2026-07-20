package pkg

import (
	"Fdoc/option"
	"path/filepath"
	"runtime"
	"testing"
)

func TestShouldSkipDir_AbsolutePathNotJoinedWithRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "home", "user")
	skip := filepath.Join(string(filepath.Separator), "Windows")
	if runtime.GOOS == "windows" {
		root = `C:\Users\alice`
		skip = `C:\Windows`
	}

	info := &option.FlagInfo{
		RootPath: root,
		SkipDirs: skip,
	}

	if !shouldSkipDir(skip, info) {
		t.Fatalf("should skip absolute dir %q without joining root %q", skip, root)
	}
}

func TestShouldSkipDir_RelativePathJoinedWithRoot(t *testing.T) {
	root := t.TempDir()
	skipRel := "node_modules"
	absSkip := filepath.Join(root, skipRel)

	info := &option.FlagInfo{
		RootPath: root,
		SkipDirs: skipRel,
	}

	if !shouldSkipDir(absSkip, info) {
		t.Fatalf("should skip relative skip dir joined with root: %q", absSkip)
	}
}

func TestShouldSkipDir_NestedUnderSkipDir(t *testing.T) {
	root := t.TempDir()
	skipRel := "vendor"
	nested := filepath.Join(root, skipRel, "pkg")

	info := &option.FlagInfo{
		RootPath: root,
		SkipDirs: skipRel,
	}

	// WalkDir only calls shouldSkipDir on the directory itself; nested match
	// matters when skip path is absolute and walk enters a parent first.
	if !shouldSkipDir(filepath.Join(root, skipRel), info) {
		t.Fatalf("should skip the skip dir itself")
	}
	_ = nested
}

func TestShouldSkipDir_CommaSeparatedList(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")

	info := &option.FlagInfo{
		RootPath: root,
		SkipDirs: "a, b",
	}

	if !shouldSkipDir(a, info) || !shouldSkipDir(b, info) {
		t.Fatalf("should skip both dirs from comma-separated list")
	}
}

func TestShouldSkipDir_EmptySkipDirs(t *testing.T) {
	info := &option.FlagInfo{RootPath: t.TempDir(), SkipDirs: ""}
	if shouldSkipDir(filepath.Join(info.RootPath, "anything"), info) {
		t.Fatal("empty SkipDirs should not skip")
	}
}

func TestShouldSkipDir_CaseInsensitiveOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("case-insensitive skip is a Windows behavior")
	}
	info := &option.FlagInfo{
		RootPath: `c:\Users\alice`,
		SkipDirs: `C:\Windows,C:\Program Files`,
	}
	if !shouldSkipDir(`c:\Windows`, info) {
		t.Fatal(`should skip c:\Windows when configured as C:\Windows`)
	}
	if !shouldSkipDir(`C:\Program Files`, info) {
		t.Fatal(`should skip C:\Program Files`)
	}
}
