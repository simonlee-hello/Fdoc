package upload

import (
	"fmt"
	"time"

	"uploader/route"
)

// Options mirrors CLI upload settings.
type Options struct {
	Backend          string // empty = auto
	Force            bool
	Quiet            bool
	Verbose          bool
	Encrypt          bool
	EncryptKey       string
	ProgressInterval time.Duration // 0=default 30s; <0=off
}

// Result is a successful upload.
type Result struct {
	URL     string
	Backend string
}

// File uploads path via uploader auto/pin selection.
func File(path string, opts Options) (Result, error) {
	link, backend, err := route.UploadAuto(path, route.Options{
		Backend:          opts.Backend,
		Force:            opts.Force,
		Quiet:            opts.Quiet,
		Verbose:          opts.Verbose,
		Mute:             true, // always capture link; callers decide what to print
		Encrypt:          opts.Encrypt,
		Key:              opts.EncryptKey,
		ProgressInterval: opts.ProgressInterval,
	})
	if err != nil {
		return Result{}, err
	}
	if link == "" {
		return Result{}, fmt.Errorf("upload succeeded but no download link returned")
	}
	return Result{URL: link, Backend: backend}, nil
}
