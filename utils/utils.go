// Package utils collects various services: simple operations, compression, etc.
package utils

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// DirIsAccessible verifies that directory exists and is accessible
func DirIsAccessible(filename string) error {
	_, err := os.Stat(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("Something went wrong, %v", err)
		}
	} else {
		if unix.Access(filename, unix.W_OK) != nil {
			return fmt.Errorf("'%s' is inaccessible, check access rights", filename)
		}
	}
	return nil
}
