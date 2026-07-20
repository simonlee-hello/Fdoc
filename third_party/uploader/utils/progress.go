package utils

import (
	"fmt"
	"io"
	"os"
	"time"
)

// ProgressReader wraps an io.Reader and prints a simple percentage to stderr.
type ProgressReader struct {
	r     io.Reader
	total int64
	read  int64
	last  time.Time
}

func NewProgressReader(r io.Reader, total int64) *ProgressReader {
	return &ProgressReader{r: r, total: total, last: time.Now().Add(-time.Second)}
}

func (p *ProgressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.read += int64(n)
		now := time.Now()
		if now.Sub(p.last) >= 200*time.Millisecond || err == io.EOF || (p.total > 0 && p.read >= p.total) {
			p.last = now
			if p.total > 0 {
				pct := float64(p.read) * 100 / float64(p.total)
				if pct > 100 {
					pct = 100
				}
				fmt.Fprintf(os.Stderr, "\r%.0f%% (%s/%s)", pct, formatBytes(p.read), formatBytes(p.total))
			} else {
				fmt.Fprintf(os.Stderr, "\r%s", formatBytes(p.read))
			}
		}
	}
	return n, err
}

func (p *ProgressReader) Finish() {
	if p.total > 0 {
		fmt.Fprintf(os.Stderr, "\r100%% (%s/%s)\n", formatBytes(p.total), formatBytes(p.total))
	} else {
		fmt.Fprintf(os.Stderr, "\r%s\n", formatBytes(p.read))
	}
}

func formatBytes(n int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.2fGB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.2fMB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.2fKB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

// ProgressWriter wraps an io.Writer with the same simple progress output.
type ProgressWriter struct {
	w     io.Writer
	total int64
	wrote int64
	last  time.Time
}

func NewProgressWriter(w io.Writer, total int64) *ProgressWriter {
	return &ProgressWriter{w: w, total: total, last: time.Now().Add(-time.Second)}
}

func (p *ProgressWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	if n > 0 {
		p.wrote += int64(n)
		now := time.Now()
		if now.Sub(p.last) >= 200*time.Millisecond || (p.total > 0 && p.wrote >= p.total) {
			p.last = now
			if p.total > 0 {
				pct := float64(p.wrote) * 100 / float64(p.total)
				if pct > 100 {
					pct = 100
				}
				fmt.Fprintf(os.Stderr, "\r%.0f%% (%s/%s)", pct, formatBytes(p.wrote), formatBytes(p.total))
			}
		}
	}
	return n, err
}

func (p *ProgressWriter) Finish() {
	if p.total > 0 {
		fmt.Fprintf(os.Stderr, "\r100%% (%s/%s)\n", formatBytes(p.total), formatBytes(p.total))
	}
}

// DefaultTickInterval is used when NoBar uploads request tick progress with TickEvery==0.
const DefaultTickInterval = 3 * time.Minute

// IntervalProgressReader prints a full stderr line at most once per interval
// (for headless / NoBar uploads where a spinner bar is undesirable).
type IntervalProgressReader struct {
	r        io.Reader
	label    string
	total    int64
	read     int64
	interval time.Duration
	last     time.Time
	start    time.Time
	started  bool
}

// NewIntervalProgressReader wraps r. interval<=0 uses DefaultTickInterval.
// label is a short tag such as "UPLOAD" or "ENCRYPT".
func NewIntervalProgressReader(r io.Reader, total int64, interval time.Duration, label string) *IntervalProgressReader {
	if interval <= 0 {
		interval = DefaultTickInterval
	}
	if label == "" {
		label = "UPLOAD"
	}
	now := time.Now()
	return &IntervalProgressReader{
		r:        r,
		label:    label,
		total:    total,
		interval: interval,
		last:     now,
		start:    now,
	}
}

func (p *IntervalProgressReader) Read(b []byte) (int, error) {
	if !p.started {
		p.started = true
		fmt.Fprintf(os.Stderr, "%s_PROGRESS start size=%s interval=%s\n",
			p.label, formatBytes(p.total), p.interval)
	}
	n, err := p.r.Read(b)
	if n > 0 {
		p.read += int64(n)
		now := time.Now()
		done := err == io.EOF || (p.total > 0 && p.read >= p.total)
		if done || now.Sub(p.last) >= p.interval {
			p.last = now
			p.printLine(now, done)
		}
	} else if err == io.EOF && p.total > 0 {
		p.printLine(time.Now(), true)
	}
	return n, err
}

func (p *IntervalProgressReader) Finish() {
	p.printLine(time.Now(), true)
}

func (p *IntervalProgressReader) printLine(now time.Time, done bool) {
	elapsed := now.Sub(p.start).Truncate(time.Second)
	cur := p.read
	if done && p.total > 0 {
		cur = p.total
	}
	if p.total > 0 {
		pct := float64(cur) * 100 / float64(p.total)
		if pct > 100 {
			pct = 100
		}
		status := "tick"
		if done {
			status = "done"
		}
		fmt.Fprintf(os.Stderr, "%s_PROGRESS %s %.0f%% (%s/%s) elapsed=%s\n",
			p.label, status, pct, formatBytes(cur), formatBytes(p.total), elapsed)
		return
	}
	status := "tick"
	if done {
		status = "done"
	}
	fmt.Fprintf(os.Stderr, "%s_PROGRESS %s %s elapsed=%s\n",
		p.label, status, formatBytes(cur), elapsed)
}

// IntervalProgressWriter is the writer counterpart for encrypt-to-temp.
type IntervalProgressWriter struct {
	w        io.Writer
	label    string
	total    int64
	wrote    int64
	interval time.Duration
	last     time.Time
	start    time.Time
	started  bool
}

func NewIntervalProgressWriter(w io.Writer, total int64, interval time.Duration, label string) *IntervalProgressWriter {
	if interval <= 0 {
		interval = DefaultTickInterval
	}
	if label == "" {
		label = "ENCRYPT"
	}
	now := time.Now()
	return &IntervalProgressWriter{
		w:        w,
		label:    label,
		total:    total,
		interval: interval,
		last:     now,
		start:    now,
	}
}

func (p *IntervalProgressWriter) Write(b []byte) (int, error) {
	if !p.started {
		p.started = true
		fmt.Fprintf(os.Stderr, "%s_PROGRESS start size=%s interval=%s\n",
			p.label, formatBytes(p.total), p.interval)
	}
	n, err := p.w.Write(b)
	if n > 0 {
		p.wrote += int64(n)
		now := time.Now()
		done := p.total > 0 && p.wrote >= p.total
		if done || now.Sub(p.last) >= p.interval {
			p.last = now
			p.printLine(now, done)
		}
	}
	return n, err
}

func (p *IntervalProgressWriter) Finish() {
	p.printLine(time.Now(), true)
}

func (p *IntervalProgressWriter) printLine(now time.Time, done bool) {
	elapsed := now.Sub(p.start).Truncate(time.Second)
	cur := p.wrote
	if done && p.total > 0 {
		cur = p.total
	}
	if p.total > 0 {
		pct := float64(cur) * 100 / float64(p.total)
		if pct > 100 {
			pct = 100
		}
		status := "tick"
		if done {
			status = "done"
		}
		fmt.Fprintf(os.Stderr, "%s_PROGRESS %s %.0f%% (%s/%s) elapsed=%s\n",
			p.label, status, pct, formatBytes(cur), formatBytes(p.total), elapsed)
		return
	}
	status := "tick"
	if done {
		status = "done"
	}
	fmt.Fprintf(os.Stderr, "%s_PROGRESS %s %s elapsed=%s\n",
		p.label, status, formatBytes(cur), elapsed)
}

// ResolveTickInterval returns the tick period for NoBar transfers.
// tickEvery==0 → DefaultTickInterval; tickEvery<0 → disabled (0).
func ResolveTickInterval(tickEvery time.Duration) time.Duration {
	if tickEvery < 0 {
		return 0
	}
	if tickEvery == 0 {
		return DefaultTickInterval
	}
	return tickEvery
}
