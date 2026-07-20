package scrub

import (
	"Fdoc/logx"
	"fmt"
	"os"
	"strings"
)

// Options controls what to remove.
type Options struct {
	Archive string
	Self    bool
	Temps   []string
}

// Result summarizes best-effort cleanup.
type Result struct {
	ArchiveOK   bool
	SelfOK      bool
	Partial     bool
	ArchivePath string
	SelfPath    string
	Errs        []string
}

// Run best-effort deletes archive, optional temps, and optionally the running binary.
func Run(opts Options) Result {
	var res Result
	res.ArchiveOK = true
	res.SelfOK = true

	for _, t := range opts.Temps {
		if t == "" {
			continue
		}
		if err := os.Remove(t); err != nil && !os.IsNotExist(err) {
			logx.Debug("scrub temp: %s: %v", t, err)
			res.Errs = append(res.Errs, fmt.Sprintf("temp %s: %v", t, err))
			res.Partial = true
		}
	}
	if opts.Archive != "" {
		res.ArchivePath = opts.Archive
		if err := os.Remove(opts.Archive); err != nil && !os.IsNotExist(err) {
			logx.Warning("scrub archive failed: %v", err)
			res.Errs = append(res.Errs, fmt.Sprintf("archive: %v", err))
			res.ArchiveOK = false
			res.Partial = true
		} else {
			logx.Info("scrubbed archive: %s", opts.Archive)
			fmt.Fprintf(os.Stderr, "SCRUB_OK archive=%s\n", opts.Archive)
		}
	}
	if opts.Self {
		path, err := removeSelf()
		res.SelfPath = path
		if err != nil {
			logx.Warning("scrub self failed: %v", err)
			res.Errs = append(res.Errs, fmt.Sprintf("self: %v", err))
			res.SelfOK = false
			res.Partial = true
		} else {
			logx.Info("scrubbed self: %s", path)
			fmt.Fprintf(os.Stderr, "SCRUB_OK self=%s\n", path)
		}
	}
	if res.Partial {
		logx.Warning("SCRUB_PARTIAL: %s", strings.Join(res.Errs, "; "))
		fmt.Fprintf(os.Stderr, "SCRUB_PARTIAL %s\n", strings.Join(res.Errs, "; "))
	} else if opts.Archive != "" || opts.Self {
		logx.Info("scrub complete")
	}
	return res
}
