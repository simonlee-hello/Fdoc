package route

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"uploader/apis"
	"uploader/apis/methods"
)

// DefaultProbeParallel is used by auto-upload when ProbeAll parallel is unspecified.
const DefaultProbeParallel = 4

// ProbeResult is the outcome of probing one backend.
type ProbeResult struct {
	Name    string
	OK      bool
	Latency time.Duration
	Link    string
	Err     string
	Skipped bool
}

// SetupBackend is an optional hook to apply backend-specific config before probe/upload.
// Cmd sets this for CLI flags (password, cookie, etc.).
var SetupBackend func(name string)

// ProbeRankedForUpload probes size-fitting backends and returns them sorted by latency.
// Probes run concurrently (DefaultProbeParallel); each uses isolated upload opts.
func ProbeRankedForUpload(maxSize int64, force bool) ([]*BackendInfo, error) {
	targets := SelectProbeTargetsForSize(maxSize, force)
	if len(targets) == 0 {
		return nil, fmt.Errorf("no backend available for this file size")
	}

	quiet := apis.QuietMode
	if !quiet {
		fmt.Fprintf(os.Stderr, "auto: probing %d backend(s) (size ≤ %s)...\n", len(targets), formatProbeSize(maxSize))
	}

	results, err := ProbeAll(targets, DefaultProbeParallel, 45*time.Second, !quiet)
	if err != nil {
		return nil, err
	}
	SortProbeResults(results)

	var ranked []*BackendInfo
	for _, r := range results {
		if !r.OK {
			continue
		}
		info := FindBackend(r.Name)
		if info == nil {
			continue
		}
		ranked = append(ranked, info)
	}
	if len(ranked) == 0 {
		return nil, fmt.Errorf("auto: no working backend (probe all failed)")
	}
	if !quiet {
		bestLat := time.Duration(0)
		for _, r := range results {
			if r.OK && r.Name == ranked[0].Name {
				bestLat = r.Latency
				break
			}
		}
		fmt.Fprintf(os.Stderr, "auto: using %s (%s)\n", ranked[0].Name, FormatLatency(bestLat))
	}
	return ranked, nil
}

func formatProbeSize(n int64) string {
	if n <= 0 {
		return "?"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// SelectProbeTargets selects backends by name list, or all ok backends when names empty.
func SelectProbeTargets(names []string, all bool) []BackendInfo {
	if len(names) > 0 {
		var out []BackendInfo
		for _, n := range names {
			info := FindBackend(n)
			if info == nil {
				fmt.Fprintf(os.Stderr, "unknown backend: %s\n", n)
				continue
			}
			out = append(out, *info)
		}
		return out
	}
	var out []BackendInfo
	for _, b := range backends {
		if !all && b.Status != "ok" {
			continue
		}
		out = append(out, b)
	}
	return out
}

// SelectProbeTargetsForSize returns backends that can hold maxSize.
func SelectProbeTargetsForSize(maxSize int64, force bool) []BackendInfo {
	var out []BackendInfo
	for _, b := range backends {
		if b.Status == "down" && !force {
			continue
		}
		if b.Status == "flaky" && !force {
			continue
		}
		if b.Status != "ok" && !force {
			continue
		}
		lim := b.MaxBytes()
		if maxSize > 0 && lim > 0 && maxSize > lim {
			continue
		}
		out = append(out, b)
	}
	return out
}

// ProbeAll probes targets concurrently up to `parallel` workers.
// Each probe uses UploadFileOpts (no global TransferConfig / Stdout mutation),
// so timed-out probes need not be joined before returning.
// On return, ClearHTTPTimeoutOverride runs so callers can safely start a real
// upload with the default HTTPTimeout even if late probe goroutines linger.
func ProbeAll(targets []BackendInfo, parallel int, timeout time.Duration, printLive bool) ([]ProbeResult, error) {
	defer methods.ClearHTTPTimeoutOverride()

	if parallel < 1 {
		parallel = 1
	}
	if parallel > len(targets) && len(targets) > 0 {
		parallel = len(targets)
	}
	probeFile, err := writeProbeFile()
	if err != nil {
		return nil, err
	}
	defer os.Remove(probeFile)

	// Mute once for the whole batch so PostUpload EmitLink stays quiet under concurrency.
	oldMute := apis.MuteMode
	apis.MuteMode = true
	defer func() { apis.MuteMode = oldMute }()

	results := make([]ProbeResult, len(targets))
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup
	var printMu sync.Mutex

	for i, info := range targets {
		i, info := i, info
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = probeOne(info, probeFile, timeout)
			if !printLive {
				return
			}
			status := "FAIL"
			if results[i].OK {
				status = "OK"
			}
			if results[i].Skipped {
				status = "SKIP"
			}
			extra := results[i].Err
			if results[i].OK {
				extra = ShortLink(results[i].Link)
			}
			printMu.Lock()
			fmt.Fprintf(os.Stderr, "%-4s %-6s %8s  %s\n", status, results[i].Name, FormatLatency(results[i].Latency), extra)
			printMu.Unlock()
		}()
	}
	wg.Wait()
	return results, nil
}

// SortProbeResults sorts successes first, then by latency.
func SortProbeResults(results []ProbeResult) {
	sort.SliceStable(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.OK != b.OK {
			return a.OK
		}
		if a.Skipped != b.Skipped {
			return !a.Skipped
		}
		if a.OK && b.OK {
			return a.Latency < b.Latency
		}
		return a.Name < b.Name
	})
}

func writeProbeFile() (string, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("uploader-probe-%d.txt", time.Now().UnixNano()))
	return path, os.WriteFile(path, []byte("uploader probe\n"), 0644)
}

func probeOne(info BackendInfo, file string, timeout time.Duration) ProbeResult {
	res := ProbeResult{Name: info.Name}
	if info.Status == "down" {
		res.Skipped = true
		res.Err = info.Note
		return res
	}

	type outcome struct {
		link    string
		err     error
		latency time.Duration
	}
	ch := make(chan outcome, 1)
	go func() {
		methods.AcquireHTTPTimeout(timeout)
		defer methods.ReleaseHTTPTimeout()
		start := time.Now()
		link, err := probeUpload(info, file)
		ch <- outcome{link: link, err: err, latency: time.Since(start)}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-timer.C:
		res.Latency = timeout
		res.Err = fmt.Sprintf("timeout >%.0fs", timeout.Seconds())
		// Do not join: probeUpload no longer mutates process-global upload state.
		// A late finish only fills the buffered channel and exits.
		return res
	case out := <-ch:
		res.Latency = out.latency
		if out.err != nil {
			res.Err = TruncateErr(out.err.Error(), 72)
			return res
		}
		res.OK = true
		res.Link = out.link
		return res
	}
}

func probeUpload(info BackendInfo, file string) (string, error) {
	if SetupBackend != nil {
		SetupBackend(info.Name)
	}
	return apis.UploadFileOpts(file, info.Backend, apis.FileUploadOpts{
		MaxBytes:    info.MaxBytes(),
		BackendName: info.Name,
		NoBar:       true,
		TickEvery:   -1, // never spam interval progress during probes
	})
}

// FormatLatency formats a duration for display.
func FormatLatency(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// ShortLink truncates a URL for display.
func ShortLink(link string) string {
	link = strings.TrimSpace(link)
	if link == "" {
		return "(no link)"
	}
	if len(link) > 64 {
		return link[:61] + "..."
	}
	return link
}

// TruncateErr shortens an error string.
func TruncateErr(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
