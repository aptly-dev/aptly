// +build freebsd

package console

// RunningOnTerminal checks whether stdout is terminal
//
// Stub for FreeBSD, until in go1.3 terminal.IsTerminal would start working for FreeBSD
func RunningOnTerminal() bool {
	return false
}
