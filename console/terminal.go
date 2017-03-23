package console

import (
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

// RunningOnTerminal checks whether stdout is terminal
func RunningOnTerminal() bool {
	return terminal.IsTerminal(syscall.Stdout)
}
