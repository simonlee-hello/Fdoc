package utils

import "testing"

func TestBase58Encode(t *testing.T) {
	if got := Base58Encode([]byte("hello")); got != "Cn8eVZg" {
		t.Fatalf("hello: got %q", got)
	}
	hex := []byte("7ceb0b114c8719bf3ac178a9e8ebb7f7")
	want := "4jDL8D8X1p6q7mBMiR2MfjMCBHWu7hapwC2gLL2jFs7c"
	if got := Base58Encode(hex); got != want {
		t.Fatalf("md5 hex: got %q want %q", got, want)
	}
}
