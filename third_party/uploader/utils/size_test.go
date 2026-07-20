package utils

import "testing"

func TestParseByteSize(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"", 0},
		{"none", 0},
		{"-", 0},
		{"100MB", 100 * 1024 * 1024},
		{"4GB", 4 * 1024 * 1024 * 1024},
		{"512MB", 512 * 1024 * 1024},
		{"50GB", 50 * 1024 * 1024 * 1024},
		{"5GB anon", 5 * 1024 * 1024 * 1024},
		{"25GB", 25 * 1024 * 1024 * 1024},
		{"200MB", 200 * 1024 * 1024},
		{"1KB", 1024},
		{"1024B", 1024},
	}
	for _, c := range cases {
		got, err := ParseByteSize(c.in)
		if err != nil {
			t.Fatalf("%q: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("%q: got %d want %d", c.in, got, c.want)
		}
	}
}

func TestFormatByteSize(t *testing.T) {
	if FormatByteSize(100*1024*1024) != "100.0MB" {
		t.Fatalf("got %s", FormatByteSize(100*1024*1024))
	}
	if FormatByteSize(0) != "unlimited" {
		t.Fatalf("got %s", FormatByteSize(0))
	}
}

func TestCheckUploadSize(t *testing.T) {
	if err := CheckUploadSize("a.bin", 50*1024*1024, 100*1024*1024, "tmpf"); err != nil {
		t.Fatal(err)
	}
	err := CheckUploadSize("a.bin", 150*1024*1024, 100*1024*1024, "tmpf")
	if err == nil {
		t.Fatal("expected error")
	}
	if err := CheckUploadSize("a.bin", 999<<30, 0, "gof"); err != nil {
		t.Fatal("unlimited should allow", err)
	}
}
