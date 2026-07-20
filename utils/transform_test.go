package utils

import "testing"

func TestParseSize(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"1KB", 1024},
		{"1MB", 1024 * 1024},
		{"1GB", 1024 * 1024 * 1024},
		{"100", 100},
		{"2.5 MB", int64(2.5 * 1024 * 1024)},
		{"0", 0},
		{"", 0},
	}
	for _, c := range cases {
		got, err := ParseSize(c.in)
		if err != nil {
			t.Fatalf("ParseSize(%q) unexpected err: %v", c.in, err)
		}
		if got != c.want {
			t.Fatalf("ParseSize(%q)=%d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseSize_RejectsInvalid(t *testing.T) {
	for _, in := range []string{"abc", "10TB", "GB", "-1MB"} {
		if _, err := ParseSize(in); err == nil {
			t.Fatalf("ParseSize(%q) should fail", in)
		}
	}
}

func TestConvertStringToList_TrimsAndDropsEmpty(t *testing.T) {
	got := ConvertStringToList(" a, ,b ,c ")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestStringToMap(t *testing.T) {
	m := StringToMap("pdf, DOCX , .xlsx,")
	for _, ext := range []string{".pdf", ".docx", ".xlsx"} {
		if _, ok := m[ext]; !ok {
			t.Fatalf("missing %s in %v", ext, m)
		}
	}
	if len(m) != 3 {
		t.Fatalf("unexpected map size: %d", len(m))
	}
	if _, ok := m["..xlsx"]; ok {
		t.Fatal("should not create ..xlsx")
	}
}
