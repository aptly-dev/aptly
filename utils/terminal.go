// +build !freebsd

package utils

import (
	"code.google.com/p/go.crypto/ssh/terminal"
	"syscall"
)

func RunningOnTerminal() bool {
	return terminal.IsTerminal(syscall.Stdout)
}
