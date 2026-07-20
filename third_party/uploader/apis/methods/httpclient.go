package methods

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

var (
	// HTTPTimeout is the default per-request timeout (large uploads).
	HTTPTimeout = 10 * time.Minute
	// HTTPRetries is extra attempts after the first (rewindable bodies only).
	HTTPRetries = 2
)

var sharedTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	ForceAttemptHTTP2:     false,
	TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
	TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
	TLSHandshakeTimeout:   15 * time.Second,
	IdleConnTimeout:       60 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

// NewClient returns a client using the shared transport. timeout<=0 uses HTTPTimeout.
func NewClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = HTTPTimeout
	}
	return &http.Client{Transport: sharedTransport, Timeout: timeout}
}

// Do performs a single request with the default client.
func Do(req *http.Request) (*http.Response, error) {
	return NewClient(0).Do(req)
}

// Get performs GET with default UA.
func Get(url string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUA)
	return Do(req)
}

// DoRetry retries when the body can be rewound (bytes slice).
func DoRetry(client *http.Client, req *http.Request, body []byte) (*http.Response, error) {
	if client == nil {
		client = NewClient(0)
	}
	tries := HTTPRetries + 1
	var lastErr error
	for i := 0; i < tries; i++ {
		if body != nil {
			req.Body = io.NopCloser(bytes.NewReader(body))
			req.ContentLength = int64(len(body))
		}
		resp, err := client.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if i+1 < tries {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	return nil, fmt.Errorf("%w (after %d tries)", lastErr, tries)
}
