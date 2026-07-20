package scrub

import (
	"Fdoc/logx"
	"os"
)

// Options controls what to remove.
type Options struct {
	Archive string
	Self    bool
	Temps   []string
}

// Run best-effort deletes archive, optional temps, and optionally the running binary.
func Run(opts Options) {
	for _, t := range opts.Temps {
		if t == "" {
			continue
		}
		if err := os.Remove(t); err != nil && !os.IsNotExist(err) {
			logx.Debug("scrub temp: %s: %v", t, err)
		}
	}
	if opts.Archive != "" {
		if err := os.Remove(opts.Archive); err != nil && !os.IsNotExist(err) {
			logx.Warning("scrub archive: %v", err)
		} else {
			logx.Debug("scrubbed archive: %s", opts.Archive)
		}
	}
	if opts.Self {
		removeSelf()
	}
}
