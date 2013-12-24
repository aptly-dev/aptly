package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// GpgDetachedSign signs file with detached signature in ASCII format
func GpgDetachedSign(source string, destination string) error {
	fmt.Printf("v = %#v\n", strings.Join([]string{"gpg", "-o", destination, "--armor", "--detach-sign", source}, " "))
	cmd := exec.Command("gpg", "-o", destination, "--armor", "--yes", "--detach-sign", source)
	return cmd.Run()
}

// GpgClearSign clear-signs the file
func GpgClearSign(source string, destination string) error {
	cmd := exec.Command("gpg", "-o", destination, "--yes", "--clearsign", source)
	return cmd.Run()
}
