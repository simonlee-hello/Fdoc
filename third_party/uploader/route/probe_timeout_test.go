package route

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"uploader/apis"
)

type slowBackend struct {
	apis.Backend
	delay time.Duration
}

func (s *slowBackend) DoUpload(string, int64, io.Reader) error {
	time.Sleep(s.delay)
	return nil
}

func (s *slowBackend) PostUpload(string, int64) (string, error) {
	return "https://example.com/slow", nil
}

func (s *slowBackend) LinkMatcher(string) bool { return true }

func TestProbeOneTimeoutDoesNotBlock(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "p.txt")
	if err := os.WriteFile(tmp, []byte("uploader probe\n"), 0644); err != nil {
		t.Fatal(err)
	}
	info := BackendInfo{
		Name:    "slow",
		Status:  "ok",
		Limit:   "10MB",
		Backend: &slowBackend{delay: 3 * time.Second},
	}
	start := time.Now()
	res := probeOne(info, tmp, 200*time.Millisecond)
	elapsed := time.Since(start)
	if res.OK {
		t.Fatal("expected timeout failure")
	}
	if elapsed > time.Second {
		t.Fatalf("timeout path blocked too long: %v (probeMu join?)", elapsed)
	}
	if res.Err == "" {
		t.Fatal("expected timeout error string")
	}
}

func TestProbeAllParallelFasterThanSerial(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "p.txt")
	if err := os.WriteFile(tmp, []byte("uploader probe\n"), 0644); err != nil {
		t.Fatal(err)
	}
	delay := 300 * time.Millisecond
	targets := make([]BackendInfo, 3)
	for i := range targets {
		targets[i] = BackendInfo{
			Name:    "b",
			Status:  "ok",
			Limit:   "10MB",
			Backend: &slowBackend{delay: delay},
		}
		targets[i].Name = string(rune('a' + i))
	}

	start := time.Now()
	results, err := ProbeAll(targets, 3, 2*time.Second, false)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	ok := 0
	for _, r := range results {
		if r.OK {
			ok++
		}
	}
	if ok != 3 {
		t.Fatalf("want 3 ok, got %d (%v)", ok, results)
	}
	// Serial would be ~900ms; parallel should be closer to ~300ms.
	if elapsed > 700*time.Millisecond {
		t.Fatalf("probe looks serial: elapsed=%v", elapsed)
	}
}
