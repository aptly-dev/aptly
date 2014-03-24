package utils

import (
	"fmt"
)

// HumanBytes converts bytes to human readable string
func HumanBytes(i int64) (result string) {
	switch {
	case i > (512 * 1024 * 1024 * 1024):
		result = fmt.Sprintf("%#.02f TiB", float64(i)/1024/1024/1024/1024)
	case i > (512 * 1024 * 1024):
		result = fmt.Sprintf("%#.02f GiB", float64(i)/1024/1024/1024)
	case i > (512 * 1024):
		result = fmt.Sprintf("%#.02f MiB", float64(i)/1024/1024)
	case i > 512:
		result = fmt.Sprintf("%#.02f KiB", float64(i)/1024)
	default:
		result = fmt.Sprintf("%d B", i)
	}
	return
}
