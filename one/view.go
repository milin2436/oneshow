package one

import (
	"bytes"
	"fmt"
	"strconv"
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
