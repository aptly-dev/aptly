package utils

import (
	"os/exec"
)

// Signer interface describes facility implementing signing of files
type Signer interface {
	SetKey(keyRef string)
	DetachedSign(source string, destination string) error
	ClearSign(source string, destination string) error
}

// Test interface
var (
	_ Signer = &GpgSigner{}
)

// GpgSigner is implementation of Signer interface using gpg
type GpgSigner struct {
	keyRef string
}

// SetKey sets key ID to use when signing files
func (g *GpgSigner) SetKey(keyRef string) {
	g.keyRef = keyRef
}

// DetachedSign signs file with detached signature in ASCII format
func (g *GpgSigner) DetachedSign(source string, destination string) error {
	args := []string{"-o", destination, "--armor", "--yes"}
	if g.keyRef != "" {
		args = append(args, "-u", g.keyRef)
	}
	args = append(args, "--detach-sign", source)
	cmd := exec.Command("gpg", args...)
	return cmd.Run()
}

// ClearSign clear-signs the file
func (g *GpgSigner) ClearSign(source string, destination string) error {
	args := []string{"-o", destination, "--yes"}
	if g.keyRef != "" {
		args = append(args, "-u", g.keyRef)
	}
	args = append(args, "--clearsign", source)
	cmd := exec.Command("gpg", args...)
	return cmd.Run()
}
