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
