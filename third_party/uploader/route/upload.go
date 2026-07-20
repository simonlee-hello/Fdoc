package route

import (
	"fmt"
	"os"
	"strings"

	"uploader/apis"
)

// Options controls UploadAuto / UploadWithOptions.
type Options struct {
	// Backend pins a single backend. Empty means auto probe + failover.
	Backend string
	Force   bool
	Quiet   bool
	Mute    bool
	Encrypt bool
	Key     string
	// OnSuccess is called with the backend name after a successful upload (e.g. save last-backend).
	OnSuccess func(backendName string)
}

// UploadAuto uploads a single file. When opts.Backend is empty, probes and failovers by file size.
// Returns the download link and the backend name that succeeded.
func UploadAuto(path string, opts Options) (link, backendName string, err error) {
	return UploadWithOptions([]string{path}, opts)
}

// UploadWithOptions uploads one or more paths with auto or pinned backend selection.
func UploadWithOptions(files []string, opts Options) (link, backendName string, err error) {
	oldQuiet, oldMute := apis.QuietMode, apis.MuteMode
	oldCfg := *apis.TransferConfig()
	defer func() {
		apis.QuietMode = oldQuiet
		apis.MuteMode = oldMute
		*apis.TransferConfig() = oldCfg
	}()

	apis.QuietMode = opts.Quiet
	apis.MuteMode = opts.Mute || opts.Quiet
	cfg := apis.TransferConfig()
	cfg.NoBarMode = opts.Mute || opts.Quiet
	cfg.CryptoMode = opts.Encrypt
	cfg.CryptoKey = opts.Key

	maxSize, err := EstimateMaxSize(files)
	if err != nil {
		return "", "", err
	}

	auto := strings.TrimSpace(opts.Backend) == ""
	var candidates []*BackendInfo
	if auto {
		candidates, err = ProbeRankedForUpload(maxSize, opts.Force)
		if err != nil {
			return "", "", err
		}
	} else {
		info := FindBackend(opts.Backend)
		if info == nil {
			return "", "", fmt.Errorf("unknown backend %q", opts.Backend)
		}
		if err := backendAllowed(info, opts.Force); err != nil {
			return "", "", err
		}
		lim := info.MaxBytes()
		if maxSize > 0 && lim > 0 && maxSize > lim {
			return "", "", fmt.Errorf("%s exceeds backend %s limit", opts.Backend, info.Name)
		}
		candidates = []*BackendInfo{info}
	}

	if len(candidates) == 0 {
		return "", "", fmt.Errorf("no backend available for this file size")
	}

	var lastErr error
	for i, info := range candidates {
		if !apis.QuietMode && auto && i > 0 {
			fmt.Fprintf(os.Stderr, "retry backend %s...\n", info.Name)
		}
		setupUploadFor(info)
		if len(files) == 1 {
			link, lastErr = uploadFileQuiet(files[0], info.Backend, apis.MuteMode)
		} else {
			lastErr = apis.Upload(files, info.Backend)
			link = "" // multi-file: links printed by Upload when MuteMode
		}
		if lastErr == nil {
			if opts.OnSuccess != nil {
				opts.OnSuccess(info.Name)
			}
			return link, info.Name, nil
		}
		if !auto {
			return "", "", lastErr
		}
		if !apis.QuietMode {
			fmt.Fprintf(os.Stderr, "backend %s failed: %v\n", info.Name, lastErr)
		}
	}
	return "", "", lastErr
}

// uploadFileQuiet runs UploadFile while swallowing provider PostUpload stdout noise.
func uploadFileQuiet(path string, backend apis.BaseBackend, mute bool) (string, error) {
	if !mute {
		return apis.UploadFile(path, backend)
	}
	old := os.Stdout
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return apis.UploadFile(path, backend)
	}
	os.Stdout = devNull
	defer func() {
		os.Stdout = old
		_ = devNull.Close()
	}()
	return apis.UploadFile(path, backend)
}

func backendAllowed(info *BackendInfo, force bool) error {
	if info == nil {
		return fmt.Errorf("unknown backend")
	}
	switch info.Status {
	case "down":
		if !force {
			return fmt.Errorf("backend %s is down (%s); use -force to try anyway", info.Name, info.Note)
		}
	case "flaky":
		if !force {
			return fmt.Errorf("backend %s is flaky (%s); use -force to try anyway", info.Name, info.Note)
		}
	}
	return nil
}

func setupUploadFor(info *BackendInfo) {
	if SetupBackend != nil {
		SetupBackend(info.Name)
	}
	cfg := apis.TransferConfig()
	cfg.MaxBytes = info.MaxBytes()
	cfg.BackendName = info.Name
	apis.SizeHint = func(size int64) string {
		alts := BackendsFitting(size)
		var filtered []string
		for _, a := range alts {
			if a != info.Name {
				if b := FindBackend(a); b != nil && b.Status == "ok" {
					filtered = append(filtered, a)
				}
			}
		}
		if len(filtered) == 0 {
			return ""
		}
		if len(filtered) > 6 {
			filtered = filtered[:6]
		}
		return "try: -b " + strings.Join(filtered, " | -b ")
	}
}
