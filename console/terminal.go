package console

import (
	"syscall"

	"golang.org/x/term"
)

// RunningOnTerminal checks whether stdout is terminal
func RunningOnTerminal() bool {
	return term.IsTerminal(syscall.Stdout)
}
