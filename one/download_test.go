package one

import (
	"net/url"
	"os"
	"testing"
)

func TestParseContentRange(t *testing.T) {
	size, err := parseContentRange("bytes 0-1/707017362")
	if err != nil {
		t.Fatal(err)
	}
	if size != 707017362 {
		t.Fatalf("size = %d, want 707017362", size)
	}

	if _, err := parseContentRange("no-slash-here"); err == nil {
		t.Fatal("expected error for a header without '/'")
	}
	if _, err := parseContentRange("bytes 0-1/not-a-number"); err == nil {
		t.Fatal("expected error for a non-numeric size")
	}
}

func TestRecordAndReadFilePosition(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "pos-*")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := recordFilePosition(f, 123456); err != nil {
		t.Fatal(err)
	}
	got, err := readFilePosition(f)
	if err != nil {
		t.Fatal(err)
	}
	if got != 123456 {
		t.Fatalf("read position = %d, want 123456", got)
	}

	// a large 64-bit value must round-trip
	if err := recordFilePosition(f, 1<<40); err != nil {
		t.Fatal(err)
	}
	got, err = readFilePosition(f)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1<<40 {
		t.Fatalf("read position = %d, want %d", got, int64(1)<<40)
	}
}

func TestGetDownloadFileName(t *testing.T) {
	u, _ := url.Parse("https://example.com/path/to/file.bin")

	if got := GetDownloadFileName(u, "", ""); got != "file.bin" {
		t.Fatalf("default name = %q, want file.bin", got)
	}
	if got := GetDownloadFileName(u, "custom.txt", ""); got != "custom.txt" {
		t.Fatalf("explicit name = %q, want custom.txt", got)
	}
	// Content-Disposition wins over an explicit name
	if got := GetDownloadFileName(u, "custom.txt", `attachment; filename="cd.bin"`); got != "cd.bin" {
		t.Fatalf("disposition name = %q, want cd.bin", got)
	}

	u2, _ := url.Parse("https://example.com/")
	if got := GetDownloadFileName(u2, "", ""); got != "index.html" {
		t.Fatalf("root name = %q, want index.html", got)
	}
}
