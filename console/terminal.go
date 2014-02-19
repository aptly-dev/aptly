// +build !freebsd

package console

import (
	"code.google.com/p/go.crypto/ssh/terminal"
	"syscall"
)

// RunningOnTerminal checks whether stdout is terminal
func RunningOnTerminal() bool {
	return terminal.IsTerminal(syscall.Stdout)
}
