package apis_test

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"uploader/apis"
)

type mockBackend struct {
	apis.Backend
	delay   time.Duration
	link    string
	uploads atomic.Int32
}

func (m *mockBackend) DoUpload(string, int64, io.Reader) error {
	m.uploads.Add(1)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return nil
}

func (m *mockBackend) PostUpload(string, int64) (string, error) {
	apis.EmitLink(m.link)
	return m.link, nil
}

func (m *mockBackend) LinkMatcher(string) bool { return true }

func TestUploadFileOptsDoesNotMutateGlobalConfig(t *testing.T) {
	cfg := apis.TransferConfig()
	cfg.MaxBytes = 111
	cfg.BackendName = "keep"
	cfg.NoBarMode = false
	cfg.CryptoMode = false

	tmp := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(tmp, []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	b := &mockBackend{link: "https://example.com/a"}
	_, err := apis.UploadFileOpts(tmp, b, apis.FileUploadOpts{
		MaxBytes:    999,
		BackendName: "probe",
		NoBar:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxBytes != 111 || cfg.BackendName != "keep" || cfg.NoBarMode {
		t.Fatalf("global config mutated: %+v", *cfg)
	}
}

func TestUploadFileOptsConcurrent(t *testing.T) {
	cfg := apis.TransferConfig()
	cfg.MaxBytes = 42
	cfg.BackendName = "orig"
	cfg.NoBarMode = false

	oldMute := apis.MuteMode
	apis.MuteMode = true
	defer func() { apis.MuteMode = oldMute }()

	tmp := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(tmp, []byte("probe"), 0644); err != nil {
		t.Fatal(err)
	}

	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b := &mockBackend{link: "https://example.com/x"}
			_, err := apis.UploadFileOpts(tmp, b, apis.FileUploadOpts{
				MaxBytes:    int64(1000 + i),
				BackendName: "b",
				NoBar:       true,
			})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if cfg.MaxBytes != 42 || cfg.BackendName != "orig" || cfg.NoBarMode {
		t.Fatalf("global config mutated under concurrency: %+v", *cfg)
	}
}

func TestEmitLinkRespectsMute(t *testing.T) {
	old := apis.MuteMode
	apis.MuteMode = true
	defer func() { apis.MuteMode = old }()
	apis.EmitLink("https://example.com/x")
}
