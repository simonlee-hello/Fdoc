package route

import (
	"fmt"
	"strings"

	"uploader/apis"
	fichier "uploader/apis/public/1fichier"
	"uploader/apis/public/bashupload"
	"uploader/apis/public/catbox"
	"uploader/apis/public/cnet"
	"uploader/apis/public/downloadgg"
	"uploader/apis/public/gofile"
	"uploader/apis/public/litterbox"
	"uploader/apis/public/null"
	"uploader/apis/public/tempsh"
	tempfiles "uploader/apis/public/tmpfiles"
	"uploader/apis/public/wenshushu"
	"uploader/utils"
)

// BackendInfo describes a registered upload backend.
type BackendInfo struct {
	Name    string
	Aliases []string
	Site    string
	Limit   string
	Status  string // ok | flaky | down
	Note    string
	Backend apis.BaseBackend
}

// MaxBytes returns the backend size limit, or 0 if unlimited/unknown.
func (b BackendInfo) MaxBytes() int64 {
	n, err := utils.ParseByteSize(b.Limit)
	if err != nil {
		return 0
	}
	return n
}

// Backends returns a copy of the registered backend list.
func Backends() []BackendInfo {
	out := make([]BackendInfo, len(backends))
	copy(out, backends)
	return out
}

// FindBackend looks up a backend by name or alias.
func FindBackend(name string) *BackendInfo {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return nil
	}
	for i := range backends {
		b := &backends[i]
		if strings.EqualFold(b.Name, n) {
			return b
		}
		for _, a := range b.Aliases {
			if strings.EqualFold(a, n) {
				return b
			}
		}
	}
	return nil
}

// BackendsFitting returns ok backend names that can hold size bytes.
func BackendsFitting(size int64) []string {
	var out []string
	for _, b := range backends {
		if b.Status != "ok" {
			continue
		}
		lim := b.MaxBytes()
		if lim == 0 || size <= lim {
			out = append(out, b.Name)
		}
	}
	return out
}

// FormatBackendTable renders a human-readable backend table.
func FormatBackendTable() string {
	var b strings.Builder
	fmt.Fprintf(&b, "  %-6s %-10s %-6s %-32s %s\n", "NAME", "LIMIT", "STATUS", "URL", "NOTES")
	for _, x := range backends {
		note := x.Note
		if note == "" {
			note = "-"
		}
		fmt.Fprintf(&b, "  %-6s %-10s %-6s %-32s %s\n", x.Name, x.Limit, x.Status, x.Site, note)
	}
	return b.String()
}

var backends = []BackendInfo{
	{Name: "temp", Aliases: []string{"tempsh"}, Site: "https://temp.sh/", Limit: "4GB", Status: "ok", Note: "recommended", Backend: tempsh.Backend},
	{Name: "tmpf", Aliases: []string{"tempfiles", "tmpfiles"}, Site: "https://tmpfiles.org/", Limit: "100MB", Status: "ok", Note: "needs file ext; CF may reset", Backend: tempfiles.Backend},
	{Name: "lit", Aliases: []string{"litterbox"}, Site: "https://litterbox.catbox.moe/", Limit: "1GB", Status: "ok", Note: "expires ~72h", Backend: litterbox.Backend},
	{Name: "gof", Aliases: []string{"gofile"}, Site: "https://gofile.io/", Limit: "none", Status: "ok", Note: "-s multi-file", Backend: gofile.Backend},
	{Name: "cnet", Aliases: []string{"paste"}, Site: "https://paste.c-net.org/", Limit: "50MB", Status: "ok", Note: "rate/ip sensitive", Backend: cnet.Backend},
	{Name: "gg", Aliases: []string{"downloadgg"}, Site: "https://download.gg/", Limit: "25GB", Status: "ok", Note: "", Backend: downloadgg.Backend},
	{Name: "fic", Aliases: []string{"1fichier"}, Site: "https://1fichier.com/", Limit: "300GB", Status: "ok", Note: "password/api-key", Backend: fichier.Backend},
	{Name: "wss", Aliases: []string{"wenshushu"}, Site: "https://wenshushu.cn/", Limit: "5GB anon", Status: "ok", Note: "-password", Backend: wenshushu.Backend},
	{Name: "cat", Aliases: []string{"catbox"}, Site: "https://catbox.moe/", Limit: "200MB", Status: "flaky", Note: "anon often blocked", Backend: catbox.Backend},
	{Name: "bash", Aliases: []string{"bashupload"}, Site: "https://bashupload.com/", Limit: "50GB", Status: "flaky", Note: "tls issues", Backend: bashupload.Backend},
	{Name: "nil", Aliases: []string{"null", "0x0"}, Site: "https://0x0.st/", Limit: "512MB", Status: "down", Note: "uploads disabled", Backend: null.Backend},
}
