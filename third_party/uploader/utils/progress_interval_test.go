package utils

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestResolveTickInterval(t *testing.T) {
	if got := ResolveTickInterval(0); got != DefaultTickInterval {
		t.Fatalf("0 → default: got %v", got)
	}
	if got := ResolveTickInterval(-1); got != 0 {
		t.Fatalf("negative → off: got %v", got)
	}
	want := 90 * time.Second
	if got := ResolveTickInterval(want); got != want {
		t.Fatalf("explicit: got %v want %v", got, want)
	}
}

func TestIntervalProgressReaderTicks(t *testing.T) {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()

	src := bytes.NewReader(bytes.Repeat([]byte("x"), 1000))
	pr := NewIntervalProgressReader(src, 1000, 5*time.Millisecond, "UPLOAD")
	buf := make([]byte, 100)
	for {
		_, err := pr.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond)
	}
	pr.Finish()
	_ = w.Close()
	var out bytes.Buffer
	_, _ = io.Copy(&out, r)
	s := out.String()
	if !strings.Contains(s, "UPLOAD_PROGRESS start") {
		t.Fatalf("missing start line: %q", s)
	}
	if !strings.Contains(s, "UPLOAD_PROGRESS") || !strings.Contains(s, "done") {
		t.Fatalf("missing done/tick: %q", s)
	}
}
