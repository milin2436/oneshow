package one

import "testing"

func TestFormatSize(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0.00"},
		{1023, "1023.00"},
		{1024, "1.00K"},
		{1024 * 1024, "1.00M"},
		{1024 * 1024 * 1024, "1.00G"},
		{1536, "1.50K"},
	}
	for _, c := range cases {
		if got := FormatSize(c.in); got != c.want {
			t.Errorf("FormatSize(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatPercent(t *testing.T) {
	if got := FormatPercent(50, 100); got != "50.0%" {
		t.Errorf("FormatPercent(50, 100) = %q, want 50.0%%", got)
	}
	if got := FormatPercent(1, 4); got != "25.0%" {
		t.Errorf("FormatPercent(1, 4) = %q, want 25.0%%", got)
	}
	if got := FormatPercent(0, 100); got != "0.0%" {
		t.Errorf("FormatPercent(0, 100) = %q, want 0.0%%", got)
	}
}

func TestGetOneDrivePath(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", "/"},
		{"/", "/"},
		{"/a", "/a"},
		{"/a/", "/a"},
		{"/a/b/c", "/a/b/c"},
		{"/a/b/c/", "/a/b/c"},
	}
	for _, c := range cases {
		if got := GetOneDrivePath(c.in); got != c.want {
			t.Errorf("GetOneDrivePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEncodePath(t *testing.T) {
	cases := []struct {
		path string
		all  bool
		want string
	}{
		{"/a%20b", false, "/a%2520b"}, // % is double-escaped
		{"/a?b", false, "/a%3Fb"},
		{"/a#b", false, "/a%23b"},
		{"/a b", true, "/a%20b"}, // full escape per segment
		{"/a b/c", false, "/a b/c"},
	}
	for _, c := range cases {
		got := EncodePath(c.path, c.all)
		if got != c.want {
			t.Errorf("EncodePath(%q, %v) = %q, want %q", c.path, c.all, got, c.want)
		}
	}
}

func TestURLPathEscape(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"/a/b", "/a/b"},
		{"/a b", "/a%20b"},
		{"/a/b c/d", "/a/b%20c/d"},
	}
	for _, c := range cases {
		if got := URLPathEscape(c.in); got != c.want {
			t.Errorf("URLPathEscape(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
