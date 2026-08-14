package one

import (
	"fmt"
	"net/url"
	"strings"
)

// FormatSize renders a byte count in human-readable units (K/M/G).
func FormatSize(isize int64) string {
	size := float64(isize)
	if size < 1024 {
		return fmt.Sprintf("%.2f", size)
	}
	tmp := size / 1024.0
	if tmp < 1024 {
		return fmt.Sprintf("%.2fK", tmp)
	}
	tmp = tmp / 1024.0
	if tmp < 1024 {
		return fmt.Sprintf("%.2fM", tmp)
	}
	tmp = tmp / 1024.0
	return fmt.Sprintf("%.2fG", tmp)
}

// FormatPercent renders sub/total as a percentage.
func FormatPercent(sub, total int64) string {
	fsub := float64(sub)
	ftotal := float64(total)
	return fmt.Sprintf("%.1f%%", fsub/ftotal*100.0)
}

// GetOneDrivePath normalizes a OneDrive path: empty becomes "/" and a single
// trailing slash is stripped.
func GetOneDrivePath(dirPath string) string {
	if dirPath == "" {
		dirPath = "/"
	}
	strLen := len(dirPath)
	if strLen > 1 && dirPath[strLen-1] == '/' {
		dirPath = dirPath[:strLen-1]
	}
	return dirPath
}

// EncodePath percent-escapes each path segment. When any bool arg is true, each
// segment is escaped with url.PathEscape; otherwise only the reserved characters
// %, ? and # are replaced.
func EncodePath(path string, all ...bool) string {
	seg := strings.Split(path, "/")
	toReplace := []struct {
		Src string
		Dst string
	}{
		{Src: "%", Dst: "%25"},
		{Src: "?", Dst: "%3F"},
		{Src: "#", Dst: "%23"},
	}
	for i := range seg {
		if len(all) > 0 && all[0] {
			seg[i] = url.PathEscape(seg[i])
		} else {
			for j := range toReplace {
				seg[i] = strings.ReplaceAll(seg[i], toReplace[j].Src, toReplace[j].Dst)
			}
		}
	}
	return strings.Join(seg, "/")
}

// URLPathEscape escapes a path the same way rclone does.
func URLPathEscape(in string) string {
	var u url.URL
	u.Path = in
	return u.String()
}
