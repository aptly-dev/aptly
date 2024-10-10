// Package utils collects various services: simple operations, compression, etc.
package utils

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// DirIsAccessible verifies that directory exists and is accessible
func DirIsAccessible(filename string) error {
	fileStat, err := os.Stat(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error checking directory '%s': %s", filename, err)
		}
	} else {
		if fileStat.Mode().Perm() == 0000 || unix.Access(filename, unix.W_OK) != nil {
			return fmt.Errorf("'%s' is inaccessible, check access rights", filename)
		}
	}
	return nil
}

// Remove leading '/', remove '..'
func PathSanitize(path string) (result string) {
	result = strings.Replace(path, "..", "", -1)
	result = strings.TrimLeft(result, "/")
	return
}
