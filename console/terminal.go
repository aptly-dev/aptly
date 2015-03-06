package console

import (
	"golang.org/x/crypto/ssh/terminal"
	"syscall"
)

// RunningOnTerminal checks whether stdout is terminal
func RunningOnTerminal() bool {
	return terminal.IsTerminal(syscall.Stdout)
}
