package utils

import (
	"os"
	"syscall"
)

// SameFilesystem checks whether two existing paths reside on the same
// filesystem and can thus be hardlinked
func SameFilesystem(path1, path2 string) (bool, error) {
	path1Stat, err := os.Stat(path1)
	if err != nil {
		return false, err
	}

	path2Stat, err := os.Stat(path2)
	if err != nil {
		return false, err
	}

	path1Sys := path1Stat.Sys().(*syscall.Stat_t)
	path2Sys := path2Stat.Sys().(*syscall.Stat_t)

	return path1Sys.Dev == path2Sys.Dev, nil
}
