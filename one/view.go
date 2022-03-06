package one

import (
	"fmt"
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

func ViewIsVideo(fileName string) bool {
	return true
}
