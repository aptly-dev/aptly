package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Signer interface describes facility implementing signing of files
type Signer interface {
	Init() error
	SetKey(keyRef string)
	SetKeyRing(keyring, secretKeyring string)
	SetPassphrase(passphrase, passphraseFile string)
	SetBatch(batch bool)
	DetachedSign(source string, destination string) error
	ClearSign(source string, destination string) error
}

// Verifier interface describes signature verification factility
type Verifier interface {
	InitKeyring() error
	AddKeyring(keyring string)
	VerifyDetachedSignature(signature, cleartext io.Reader) error
	IsClearSigned(clearsigned io.Reader) (bool, error)
	VerifyClearsigned(clearsigned io.Reader, showKeyTip bool) (*GpgKeyInfo, error)
	ExtractClearsigned(clearsigned io.Reader) (text *os.File, err error)
}

// Test interface
var (
	_ Signer   = &GpgSigner{}
	_ Verifier = &GpgVerifier{}
)

// GpgKey is key in GPG representation
type GpgKey string

// Matches checks two keys for equality
func (key1 GpgKey) Matches(key2 GpgKey) bool {
	if key1 == key2 {
		return true
	}

	if len(key1) == 8 && len(key2) == 16 {
		return key1 == key2[8:]
	}

	if len(key1) == 16 && len(key2) == 8 {
		return key1[8:] == key2
	}

	return false
}

// GpgKeyInfo is response from signature verification
type GpgKeyInfo struct {
	GoodKeys    []GpgKey
	MissingKeys []GpgKey
}

// GpgSigner is implementation of Signer interface using gpg
type GpgSigner struct {
	keyRef                     string
	keyring, secretKeyring     string
	passphrase, passphraseFile string
	batch                      bool
}

// SetBatch control --no-tty flag to gpg
func (g *GpgSigner) SetBatch(batch bool) {
	g.batch = batch
}

// SetKey sets key ID to use when signing files
func (g *GpgSigner) SetKey(keyRef string) {
	g.keyRef = keyRef
}

// SetKeyRing allows to set custom keyring and secretkeyring
func (g *GpgSigner) SetKeyRing(keyring, secretKeyring string) {
	g.keyring, g.secretKeyring = keyring, secretKeyring
}

// SetPassphrase sets passhprase params
func (g *GpgSigner) SetPassphrase(passphrase, passphraseFile string) {
	g.passphrase, g.passphraseFile = passphrase, passphraseFile
}

func (g *GpgSigner) gpgArgs() []string {
	args := []string{}
	if g.keyring != "" {
		args = append(args, "--no-auto-check-trustdb", "--no-default-keyring", "--keyring", g.keyring)
	}
	if g.secretKeyring != "" {
		args = append(args, "--secret-keyring", g.secretKeyring)
	}

	if g.keyRef != "" {
		args = append(args, "-u", g.keyRef)
	}

	if g.passphrase != "" || g.passphraseFile != "" {
		args = append(args, "--no-use-agent")
	}

	if g.passphrase != "" {
		args = append(args, "--passphrase", g.passphrase)
	}

	if g.passphraseFile != "" {
		args = append(args, "--passphrase-file", g.passphraseFile)
	}

	if g.batch {
		args = append(args, "--no-tty")
	}

	return args
}

// Init verifies availability of gpg & presence of keys
func (g *GpgSigner) Init() error {
	output, err := exec.Command("gpg", "--list-keys", "--dry-run", "--no-auto-check-trustdb").CombinedOutput()
	if err != nil {
		return fmt.Errorf("unable to execute gpg: %s (is gpg installed?): %s", err, string(output))
	}

	if g.keyring == "" && g.secretKeyring == "" && len(output) == 0 {
		return fmt.Errorf("looks like there are no keys in gpg, please create one (official manual: http://www.gnupg.org/gph/en/manual.html)")
	}

	return err
}

// DetachedSign signs file with detached signature in ASCII format
func (g *GpgSigner) DetachedSign(source string, destination string) error {
	fmt.Printf("Signing file '%s' with gpg, please enter your passphrase when prompted:\n", filepath.Base(source))

	args := []string{"-o", destination, "--digest-algo", "SHA256", "--armor", "--yes"}
	args = append(args, g.gpgArgs()...)
	args = append(args, "--detach-sign", source)
	cmd := exec.Command("gpg", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ClearSign clear-signs the file
func (g *GpgSigner) ClearSign(source string, destination string) error {
	fmt.Printf("Clearsigning file '%s' with gpg, please enter your passphrase when prompted:\n", filepath.Base(source))
	args := []string{"-o", destination, "--digest-algo", "SHA256", "--yes"}
	args = append(args, g.gpgArgs()...)
	args = append(args, "--clearsign", source)
	cmd := exec.Command("gpg", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GpgVerifier is implementation of Verifier interface using gpgv
type GpgVerifier struct {
	keyRings []string
}

// InitKeyring verifies that gpg is installed and some keys are trusted
func (g *GpgVerifier) InitKeyring() error {
	err := exec.Command("gpgv", "--version").Run()
	if err != nil {
		return fmt.Errorf("unable to execute gpgv: %s (is gpg installed?)", err)
	}

	if len(g.keyRings) == 0 {
		// using default keyring
		output, err := exec.Command("gpg", "--no-default-keyring", "--no-auto-check-trustdb", "--keyring", "trustedkeys.gpg", "--list-keys").Output()
		if err == nil && len(output) == 0 {
			fmt.Printf("\nLooks like your keyring with trusted keys is empty. You might consider importing some keys.\n")
			fmt.Printf("If you're running Debian or Ubuntu, it's a good idea to import current archive keys by running:\n\n")
			fmt.Printf("  gpg --no-default-keyring --keyring /usr/share/keyrings/debian-archive-keyring.gpg --export | gpg --no-default-keyring --keyring trustedkeys.gpg --import\n")
			fmt.Printf("\n(for Ubuntu, use /usr/share/keyrings/ubuntu-archive-keyring.gpg)\n\n")
		}
	}

	return nil
}

// AddKeyring adds custom keyring to GPG parameters
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

func (g *GpgVerifier) runGpgv(args []string, context string, showKeyTip bool) (*GpgKeyInfo, error) {
	args = append([]string{"--status-fd", "3"}, args...)
	cmd := exec.Command("gpgv", args...)

	tempf, err := ioutil.TempFile("", "aptly-gpg-status")
	if err != nil {
		return nil, err
	}
	defer tempf.Close()

	err = os.Remove(tempf.Name())
	if err != nil {
		return nil, err
	}

	cmd.ExtraFiles = []*os.File{tempf}

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

	cmderr := cmd.Wait()

	tempf.Seek(0, 0)

	statusr := bufio.NewScanner(tempf)

	result := &GpgKeyInfo{}

	for statusr.Scan() {
		line := strings.TrimSpace(statusr.Text())

		if strings.HasPrefix(line, "[GNUPG:] GOODSIG ") {
			result.GoodKeys = append(result.GoodKeys, GpgKey(strings.Fields(line)[2]))
		} else if strings.HasPrefix(line, "[GNUPG:] NO_PUBKEY ") {
			result.MissingKeys = append(result.MissingKeys, GpgKey(strings.Fields(line)[2]))
		}
	}

	if err = statusr.Err(); err != nil {
		return nil, err
	}

	if cmderr != nil {
		if showKeyTip && len(g.keyRings) == 0 && len(result.MissingKeys) > 0 {
			fmt.Printf("\nLooks like some keys are missing in your trusted keyring, you may consider importing them from keyserver:\n\n")

			keys := make([]string, len(result.MissingKeys))

			for i := range result.MissingKeys {
				keys[i] = string(result.MissingKeys[i])
			}

			fmt.Printf("gpg --no-default-keyring --keyring trustedkeys.gpg --keyserver keys.gnupg.net --recv-keys %s\n\n",
				strings.Join(keys, " "))

			fmt.Printf("Sometimes keys are stored in repository root in file named Release.key, to import such key:\n\n")
			fmt.Printf("wget -O - https://some.repo/repository/Release.key | gpg --no-default-keyring --keyring trustedkeys.gpg --import\n\n")
		}
		return result, fmt.Errorf("verification of %s failed: %s", context, cmderr)
	}
	return result, nil
}

// VerifyDetachedSignature verifies combination of signature and cleartext using gpgv
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
	_, err = g.runGpgv(args, "detached signature", true)
	return err
}

// IsClearSigned returns true if file contains signature
func (g *GpgVerifier) IsClearSigned(clearsigned io.Reader) (bool, error) {
	scanner := bufio.NewScanner(clearsigned)
	for scanner.Scan() {
		if strings.Index(scanner.Text(), "BEGIN PGP SIGN") != -1 {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

// VerifyClearsigned verifies clearsigned file using gpgv
func (g *GpgVerifier) VerifyClearsigned(clearsigned io.Reader, showKeyTip bool) (*GpgKeyInfo, error) {
	args := g.argsKeyrings()

	clearf, err := ioutil.TempFile("", "aptly-gpg")
	if err != nil {
		return nil, err
	}
	defer os.Remove(clearf.Name())
	defer clearf.Close()

	_, err = io.Copy(clearf, clearsigned)
	if err != nil {
		return nil, err
	}

	args = append(args, clearf.Name())
	return g.runGpgv(args, "clearsigned file", showKeyTip)
}

// ExtractClearsigned extracts cleartext from clearsigned file WITHOUT signature verification
func (g *GpgVerifier) ExtractClearsigned(clearsigned io.Reader) (text *os.File, err error) {
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

	text, err = ioutil.TempFile("", "aptly-gpg")
	if err != nil {
		return
	}
	defer os.Remove(text.Name())

	args := []string{"--no-auto-check-trustdb", "--decrypt", "--batch", "--skip-verify", "--output", "-", clearf.Name()}

	cmd := exec.Command("gpg", args...)
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
		return nil, fmt.Errorf("extraction of clearsigned file failed: %s", err)
	}

	_, err = text.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	return
}
