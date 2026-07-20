package methods

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

const defaultUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

// MultipartUpload streams a single file field named "file" to endpoint.
func MultipartUpload(config MultiPartUploadConfig) ([]byte, error) {
	if config.Debug {
		log.Printf("start upload -> %s", config.Endpoint)
	}
	return multipartUploadOnce(config)
}

func multipartUploadOnce(config MultiPartUploadConfig) ([]byte, error) {
	client := NewClient(0)

	headerBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(headerBuf)
	if _, err := writer.CreateFormFile("file", config.FileName); err != nil {
		return nil, err
	}
	headerBytes := append([]byte(nil), headerBuf.Bytes()...)
	boundary := writer.Boundary()
	// Close only to finalize writer state; closing bytes are rebuilt below for streaming.
	_ = writer.Close()

	lastBoundary := []byte("\r\n--" + boundary + "--\r\n")
	totalSize := int64(len(headerBytes)) + config.FileSize + int64(len(lastBoundary))

	partR, partW := io.Pipe()
	go func() {
		var fail error
		defer func() {
			if fail != nil {
				_ = partW.CloseWithError(fail)
				return
			}
			_ = partW.Close()
		}()
		if _, fail = partW.Write(headerBytes); fail != nil {
			return
		}
		if _, fail = io.Copy(partW, config.FileReader); fail != nil {
			return
		}
		_, fail = partW.Write(lastBoundary)
	}()

	req, err := http.NewRequest(http.MethodPost, config.Endpoint, partR)
	if err != nil {
		_ = partR.Close()
		return nil, err
	}
	req.ContentLength = totalSize
	req.Header.Set("Content-Length", strconv.FormatInt(totalSize, 10))
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	if config.Debug {
		log.Printf("header: %v", req.Header)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		msg := strings.TrimSpace(string(body))
		if strings.EqualFold(msg, "Blacklisted") || strings.Contains(strings.ToLower(msg), "blacklisted") {
			return nil, fmt.Errorf("http %d: ip blacklisted", resp.StatusCode)
		}
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, truncate(msg, 120))
	}
	if config.Debug {
		log.Printf("returns: %s", truncate(string(body), 300))
	}
	return body, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
