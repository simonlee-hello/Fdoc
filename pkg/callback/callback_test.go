package callback

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEncodeDNSChunks(t *testing.T) {
	task, chunks := EncodeDNSChunks("op42", "host1", "https://temp.sh/abc")
	if task != "op42" {
		t.Fatalf("task label: %q", task)
	}
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	for _, c := range chunks {
		if len(c) > 63 {
			t.Fatalf("chunk too long: %d", len(c))
		}
	}
	joined := strings.Join(chunks, "")
	if !strings.Contains(joined, "a") { // base32 alphabet
		t.Fatalf("unexpected encoding: %q", joined)
	}
}

func TestNotifyWebhookOK(t *testing.T) {
	var got Payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("content-type %s", ct)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res, err := Notify(context.Background(), Payload{
		TaskID: "t1",
		Host:   "h",
		URL:    "https://example.com/f",
		Size:   10,
		Files:  2,
	}, Options{Webhook: srv.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if !res.WebhookOK || res.DNSOK {
		t.Fatalf("result: %+v", res)
	}
	if got.URL != "https://example.com/f" || got.TaskID != "t1" {
		t.Fatalf("payload: %+v", got)
	}
}

func TestNotifyWebhookRejectsNonLoopbackHTTP(t *testing.T) {
	_, err := Notify(context.Background(), Payload{
		TaskID: "t1", Host: "h", URL: "https://example.com/f",
	}, Options{Webhook: "http://example.com/hook", Timeout: time.Second})
	if err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected https requirement, got %v", err)
	}
}

func TestNotifyWebhookFailDNSConfigured(t *testing.T) {
	// Webhook 500; DNS to *.invalid usually NXDOMAIN → counts as query sent for all chunks.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	res, err := Notify(context.Background(), Payload{
		TaskID: "t2",
		Host:   "h",
		URL:    "https://example.com/f",
	}, Options{
		Webhook: srv.URL,
		DNS:     "invalid",
		Timeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("expected dns failover success (NXDOMAIN), got %v", err)
	}
	if res.WebhookOK {
		t.Fatal("webhook should fail")
	}
	if !res.DNSOK {
		t.Fatal("dns should succeed when all chunks get NXDOMAIN/answer")
	}
}

func TestSendDNSRejectsEmptyTask(t *testing.T) {
	err := sendDNS(context.Background(), "example.com", Payload{
		TaskID: "!!!",
		Host:   "h",
		URL:    "https://x",
	})
	if err == nil || !strings.Contains(err.Error(), "sanitize") {
		t.Fatalf("got %v", err)
	}
}
