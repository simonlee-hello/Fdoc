package apis

import "testing"

func TestPreferEncryptedRemoteName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"out.tgz", "out.bin"},
		{"out.tar.gz", "out.bin"}, // preferTgzName first in uploadWith; this tests helper alone
		{"plain", "plain.bin"},
		{"data.bin", "data.bin"},
		{"x.encrypt", "x.bin"},
		{"report.pdf", "report.bin"},
	}
	for _, c := range cases {
		in := c.in
		if c.in == "out.tar.gz" {
			in = preferTgzName(c.in)
		}
		if got := preferEncryptedRemoteName(in); got != c.want {
			t.Fatalf("preferEncryptedRemoteName(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestPreferTgzName(t *testing.T) {
	if got := preferTgzName("a.tar.gz"); got != "a.tgz" {
		t.Fatalf("got %q", got)
	}
	if got := preferTgzName("a.tgz"); got != "a.tgz" {
		t.Fatalf("got %q", got)
	}
}
