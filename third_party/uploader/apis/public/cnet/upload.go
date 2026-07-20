package cnet

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"uploader/apis"
	"uploader/apis/methods"
)

const upload = "https://paste.c-net.org/"

var cnetLinkRe = regexp.MustCompile(`https?://paste\.c-net\.org/[A-Za-z0-9]+`)

func (b *cnet) DoUpload(name string, size int64, file io.Reader) error {
	body, err := methods.MultipartUpload(methods.MultiPartUploadConfig{
		FileSize:   size,
		FileName:   name,
		FileReader: file,
		Debug:      apis.DebugMode,
		Endpoint:   upload,
		Headers: map[string]string{
			// Prefer plain URL; HTML Success page is also parsed as fallback.
			"Accept": "text/plain",
		},
	})
	if err != nil {
		return fmt.Errorf("upload returns error: %s", err)
	}

	link, err := parseCnetLink(string(body))
	if err != nil {
		return err
	}
	b.resp = link
	return nil
}

func parseCnetLink(body string) (string, error) {
	resp := strings.TrimSpace(body)
	if strings.HasPrefix(resp, "http://") || strings.HasPrefix(resp, "https://") {
		// plain-text response may still contain trailing junk
		if m := cnetLinkRe.FindString(resp); m != "" {
			return m, nil
		}
		return strings.Fields(resp)[0], nil
	}
	if m := cnetLinkRe.FindString(resp); m != "" {
		return m, nil
	}
	return "", fmt.Errorf("upload error: cannot find download link in response: %s", truncate(resp, 180))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func (b *cnet) PostUpload(string, int64) (string, error) {
	apis.EmitLink(b.resp)
	return b.resp, nil
}
