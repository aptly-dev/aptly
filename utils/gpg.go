package utils

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Signer interface describes facility implementing signing of files
type Signer interface {
	Init() error
	SetKey(keyRef string)
	DetachedSign(source string, destination string) error
	ClearSign(source string, destination string) error
}

// Verifier interface describes signature verification factility
type Verifier interface {
	InitKeyring() error
	AddKeyring(keyring string)
	VerifyDetachedSignature(signature, cleartext io.Reader) error
	VerifyClearsigned(clearsigned io.Reader) (text *os.File, err error)
}

// Test interface
var (
	_ Signer   = &GpgSigner{}
	_ Verifier = &GpgVerifier{}
)

// GpgSigner is implementation of Signer interface using gpg
type GpgSigner struct {
	keyRef string
}

// SetKey sets key ID to use when signing files
func (g *GpgSigner) SetKey(keyRef string) {
	g.keyRef = keyRef
}

// Init verifies availability of gpg & presence of keys
func (g *GpgSigner) Init() error {
	output, err := exec.Command("gpg", "--list-keys").Output()
	if err != nil {
		return fmt.Errorf("unable to execute gpg: %s (is gpg installed?)", err)
	}

	if len(output) == 0 {
		return fmt.Errorf("looks like there are no keys in gpg, please create one (official manual: http://www.gnupg.org/gph/en/manual.html)")
	}

	return err
}

// DetachedSign signs file with detached signature in ASCII format
func (g *GpgSigner) DetachedSign(source string, destination string) error {
	fmt.Printf("Signing file '%s' with gpg, please enter your passphrase when prompted:\n", source)

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
	fmt.Printf("Clearsigning file '%s' with gpg, please enter your passphrase when prompted:\n", source)
	args := []string{"-o", destination, "--yes"}
	if g.keyRef != "" {
		args = append(args, "-u", g.keyRef)
	}
	args = append(args, "--clearsign", source)
	cmd := exec.Command("gpg", args...)
	return cmd.Run()
}

// GpgVerifier is implementation of Verifier interface using gpgv
type GpgVerifier struct {
	keyRings []string
}

func (g *GpgVerifier) InitKeyring() error {
	err := exec.Command("gpgv", "--version").Run()
	if err != nil {
		return fmt.Errorf("unable to execute gpgv: %s (is gpg installed?)", err)
	}

	if len(g.keyRings) == 0 {
		// using default keyring
		output, err := exec.Command("gpg", "--no-default-keyring", "--keyring", "trustedkeys.gpg", "--list-keys").Output()
		if err == nil && len(output) == 0 {
			fmt.Printf("\nLooks like your keyring with trusted keys is empty. You might consider importing some keys.\n")
			fmt.Printf("If you're running Debian or Ubuntu, it's a good idea to import current archive keys by running:\n\n")
			fmt.Printf("  gpg --keyring /usr/share/keyrings/debian-archive-keyring.gpg --export | gpg --no-default-keyring --keyring trustedkeys.gpg --import\n")
			fmt.Printf("\n(for Ubuntu, use /usr/share/keyrings/ubuntu-archive-keyring.gpg)\n\n")
		}
	}

	return nil
}

func (g *GpgVerifier) AddKeyring(keyring string) {
	g.keyRings = append(g.keyRings, keyring)
}

func (g *GpgVerifier) argsKeyrings() (args []string) {
	if len(g.keyRings) > 0 {
		args = make([]string, 0, 2*len(g.keyRings))
		for _, keyring := range g.keyRings {
			args = append(args, "--keyring", keyring)
		}
	} else {
		args = []string{"--keyring", "trustedkeys.gpg"}
	}
	return
}

func (g *GpgVerifier) VerifyDetachedSignature(signature, cleartext io.Reader) error {
	args := g.argsKeyrings()

	sigf, err := ioutil.TempFile("", "aptly-gpg")
	if err != nil {
		return err
	}
	defer os.Remove(sigf.Name())
	defer sigf.Close()

	_, err = io.Copy(sigf, signature)
	if err != nil {
		return err
	}

	clearf, err := ioutil.TempFile("", "aptly-gpg")
	if err != nil {
		return err
	}
	defer os.Remove(clearf.Name())
	defer clearf.Close()

	_, err = io.Copy(clearf, cleartext)
	if err != nil {
		return err
	}

	args = append(args, sigf.Name(), clearf.Name())
	cmd := exec.Command("gpgv", args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer stderr.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	buffer := &bytes.Buffer{}

	_, err = io.Copy(io.MultiWriter(os.Stderr, buffer), stderr)
	if err != nil {
		return err
	}

	matches := regexp.MustCompile("ID ([0-9A-F]{8})").FindAllStringSubmatch(buffer.String(), -1)

	err = cmd.Wait()
	if err != nil {
		if len(g.keyRings) == 0 && len(matches) > 0 {
			fmt.Printf("\nLooks like some keys are missing in your trusted keyring, you may consider importing them from keyserver:\n\n")

			keyIDs := []string{}
			for _, match := range matches {
				keyIDs = append(keyIDs, match[1])
			}
			fmt.Printf("gpg --no-default-keyring --keyring trustedkeys.gpg --keyserver keys.gnupg.net --recv-keys %s\n\n",
				strings.Join(keyIDs, " "))
		}
		return fmt.Errorf("GnuPG verification of detached signature failed: %s", err)
	}
	return nil
}

func (g *GpgVerifier) VerifyClearsigned(clearsigned io.Reader) (text *os.File, err error) {
	args := g.argsKeyrings()

	clearf, err := ioutil.TempFile("", "aptly-gpg")
	if err != nil {
		return
	}
	defer os.Remove(clearf.Name())
	defer clearf.Close()

	_, err = io.Copy(clearf, clearsigned)
	if err != nil {
		return
	}

	args = append(args, clearf.Name())
	cmd := exec.Command("gpgv", args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	defer stderr.Close()

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	buffer := &bytes.Buffer{}

	_, err = io.Copy(io.MultiWriter(os.Stderr, buffer), stderr)
	if err != nil {
		return nil, err
	}

	matches := regexp.MustCompile("ID ([0-9A-F]{8})").FindAllStringSubmatch(buffer.String(), -1)

	err = cmd.Wait()
	if err != nil {
		if len(g.keyRings) == 0 && len(matches) > 0 {
			fmt.Printf("\nLooks like some keys are missing in your trusted keyring, you may consider importing them from keyserver:\n\n")

			keyIDs := []string{}
			for _, match := range matches {
				keyIDs = append(keyIDs, match[1])
			}
			fmt.Printf("gpg --no-default-keyring --keyring trustedkeys.gpg --keyserver keys.gnupg.net --recv-keys %s\n\n",
				strings.Join(keyIDs, " "))
		}
		return nil, fmt.Errorf("GnuPG verification of clearsigned file failed: %s", err)
	}

	text, err = ioutil.TempFile("", "aptly-gpg")
	if err != nil {
		return
	}
	defer os.Remove(text.Name())

	args = []string{"--no-default-keyring"}
	args = append(args, g.argsKeyrings()...)
	args = append(args, "--decrypt", "--batch", "--output", "-", clearf.Name())

	cmd = exec.Command("gpg", args...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer stdout.Close()

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(text, stdout)
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()

	if err != nil {
		return nil, fmt.Errorf("GnuPG extraction of clearsigned file failed: %s", err)
	}

	_, err = text.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	return
}
