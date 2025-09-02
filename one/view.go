package one

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

func ViewHumanShow(isize int64) string {
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
func ViewPercent(sub, total int64) string {
	fsub := float64(sub)
	ftotal := float64(total)
	return fmt.Sprintf("%.1f%%", fsub/ftotal*100.0)
}

func EscapeJSONString(input string) string {
	var buffer bytes.Buffer

	for _, char := range input {
		switch char {
		case '"':
			buffer.WriteString(`\"`)
		case '\\':
			buffer.WriteString(`\\`)
		case '\b':
			buffer.WriteString(`\b`)
		case '\f':
			buffer.WriteString(`\f`)
		case '\n':
			buffer.WriteString(`\n`)
		case '\r':
			buffer.WriteString(`\r`)
		case '\t':
			buffer.WriteString(`\t`)
		default:
			if char < 0x20 {
				buffer.WriteString(`\u00`)
				buffer.WriteString(strconv.FormatInt(int64(char), 16))
			} else {
				buffer.WriteRune(char)
			}
		}
	}

	return buffer.String()
}

func GetOnedrivePath(dirPath string) string {
	if dirPath == "" {
		dirPath = "/"
	}
	strLen := len(dirPath)
	if strLen > 1 && dirPath[strLen-1] == '/' {
		dirPath = dirPath[:strLen-1]
	}
	return dirPath
}

//alist EncodePath
func EncodePath(path string, all ...bool) string {
	seg := strings.Split(path, "/")
	toReplace := []struct {
		Src string
		Dst string
	}{
		{Src: "%", Dst: "%25"},
		{"%", "%25"},
		{"?", "%3F"},
		{"#", "%23"},
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

//rclone PathEscape
func URLPathEscape(in string) string {
	var u url.URL
	u.Path = in
	return u.String()
}

// Abbreviate function shortens the input string to the specified length
// while keeping as many characters from the start and end as possible.
func Abbreviate(s string, length int) string {
	if utf8.RuneCountInString(s) <= length {
		return s
	}

	if length <= 3 {
		// If the target length is too small to fit any characters plus "...", return just "..."
		return "..."
	}

	// Calculate the number of characters to keep from the start and end
	halfLen := (length - 3) / 2
	runes := []rune(s)
	start := runes[:halfLen]
	end := runes[len(runes)-halfLen:]

	// If length is odd, we need one extra character at the start
	if (length-3)%2 != 0 {
		start = append(start, runes[halfLen])
	}

	return fmt.Sprintf("%s...%s", string(start), string(end))
}
