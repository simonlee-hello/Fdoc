package callback

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Payload is sent via webhook JSON and/or DNS exfil.
type Payload struct {
	TaskID    string `json:"task_id"`
	Host      string `json:"host"`
	URL       string `json:"url"`
	Backend   string `json:"backend"`
	Archive   string `json:"archive"`
	Size      int64  `json:"size"`
	Files     int    `json:"files"`
	Truncated bool   `json:"truncated"`
	TS        int64  `json:"ts"`
}

// Options configures Notify channels.
type Options struct {
	Webhook string
	DNS     string
	Timeout time.Duration
}

// Result describes which channels succeeded.
type Result struct {
	WebhookOK bool
	DNSOK     bool
}

// Notify sends payload: webhook first, then DNS on failure or if webhook unset.
// DNS success requires every chunk query to be sent (NXDOMAIN counts; timeout does not).
func Notify(ctx context.Context, p Payload, opts Options) (Result, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}
	if p.TS == 0 {
		p.TS = time.Now().Unix()
	}

	var res Result
	var errs []string

	if opts.Webhook != "" {
		if err := sendWebhook(ctx, opts.Webhook, opts.Timeout, p); err != nil {
			errs = append(errs, "webhook: "+err.Error())
		} else {
			res.WebhookOK = true
			return res, nil
		}
	}

	if opts.DNS != "" {
		if err := sendDNS(ctx, opts.DNS, p); err != nil {
			errs = append(errs, "dns: "+err.Error())
		} else {
			res.DNSOK = true
			return res, nil
		}
	}

	if len(errs) == 0 {
		return res, fmt.Errorf("no callback channel configured")
	}
	return res, fmt.Errorf("callback failed: %s", strings.Join(errs, "; "))
}

func sendWebhook(ctx context.Context, rawURL string, timeout time.Duration, p Payload) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		// ok
	case "http":
		if !isLoopbackHost(u.Hostname()) {
			return fmt.Errorf("webhook must use https (http only allowed for loopback)")
		}
	default:
		return fmt.Errorf("webhook scheme must be https")
	}

	body, err := json.Marshal(p)
	if err != nil {
		return err
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Fdoc/callback")

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// DNS wire format: base32(no pad) of "task_id|host|url", chunked into labels.
// Query: <seq>-<total>-<chunk>.<task_id>.<dns-base>
func sendDNS(ctx context.Context, base string, p Payload) error {
	base = strings.TrimSuffix(strings.TrimSpace(base), ".")
	if base == "" {
		return fmt.Errorf("empty dns base")
	}
	task := sanitizeDNSLabel(p.TaskID)
	if task == "" {
		return fmt.Errorf("task_id %q is empty after DNS sanitize", p.TaskID)
	}

	raw := p.TaskID + "|" + p.Host + "|" + p.URL
	encoded := strings.TrimRight(base32.StdEncoding.EncodeToString([]byte(raw)), "=")
	encoded = strings.ToLower(encoded)

	chunks := chunkString(encoded, 40)
	if len(chunks) == 0 {
		return fmt.Errorf("empty dns payload")
	}

	resolver := net.DefaultResolver
	var failed []string
	for i, chunk := range chunks {
		if len(chunk) > 63 {
			return fmt.Errorf("dns chunk %d label too long (%d)", i, len(chunk))
		}
		name := fmt.Sprintf("%d-%d-%s.%s.%s", i, len(chunks), chunk, task, base)
		if len(name) > 253 {
			return fmt.Errorf("dns name too long (%d): chunk %d", len(name), i)
		}
		lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, err := resolver.LookupHost(lookupCtx, name)
		cancel()
		if err == nil {
			continue
		}
		// NXDOMAIN / no such host still means the query left this host.
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
			continue
		}
		failed = append(failed, fmt.Sprintf("%d:%v", i, err))
	}
	if len(failed) > 0 {
		return fmt.Errorf("dns exfil incomplete (%d/%d chunks failed): %s",
			len(failed), len(chunks), strings.Join(failed, "; "))
	}
	return nil
}

func chunkString(s string, size int) []string {
	if size <= 0 {
		size = 40
	}
	var out []string
	for len(s) > 0 {
		if len(s) <= size {
			out = append(out, s)
			break
		}
		out = append(out, s[:size])
		s = s[size:]
	}
	return out
}

func sanitizeDNSLabel(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 40 {
		out = out[:40]
	}
	return out
}

// EncodeDNSChunks exposes chunking for tests.
func EncodeDNSChunks(taskID, host, url string) (taskLabel string, chunks []string) {
	taskLabel = sanitizeDNSLabel(taskID)
	raw := taskID + "|" + host + "|" + url
	encoded := strings.TrimRight(base32.StdEncoding.EncodeToString([]byte(raw)), "=")
	encoded = strings.ToLower(encoded)
	return taskLabel, chunkString(encoded, 40)
}
