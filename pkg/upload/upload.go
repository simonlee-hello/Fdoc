package upload

import (
	"fmt"

	"uploader/route"
)

// Options mirrors CLI upload settings.
type Options struct {
	Backend    string // empty = auto
	Force      bool
	Quiet      bool
	Encrypt    bool
	EncryptKey string
}

// Result is a successful upload.
type Result struct {
	URL     string
	Backend string
}

// File uploads path via uploader auto/pin selection.
func File(path string, opts Options) (Result, error) {
	link, backend, err := route.UploadAuto(path, route.Options{
		Backend: opts.Backend,
		Force:   opts.Force,
		Quiet:   opts.Quiet,
		Mute:    true, // always capture link; callers decide what to print
		Encrypt: opts.Encrypt,
		Key:     opts.EncryptKey,
	})
	if err != nil {
		return Result{}, err
	}
	if link == "" {
		return Result{}, fmt.Errorf("upload succeeded but no download link returned")
	}
	return Result{URL: link, Backend: backend}, nil
}
