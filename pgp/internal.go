package pgp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
)

// Test interface
var (
	_ Signer   = &GoSigner{}
	_ Verifier = &GoVerifier{}
)

// GoSigner is implementation of Signer interface using Go internal OpenPGP library
type GoSigner struct {
	keyRef                     string
	keyring, secretKeyring     string
	passphrase, passphraseFile string
	batch                      bool
}

// SetBatch controls whether we allowed to interact with user
func (g *GoSigner) SetBatch(batch bool) {
	g.batch = batch
}

// SetKey sets key ID to use when signing files
func (g *GoSigner) SetKey(keyRef string) {
	g.keyRef = keyRef
}

// SetKeyRing allows to set custom keyring and secretkeyring
func (g *GoSigner) SetKeyRing(keyring, secretKeyring string) {
	g.keyring, g.secretKeyring = keyring, secretKeyring
}

// SetPassphrase sets passhprase params
func (g *GoSigner) SetPassphrase(passphrase, passphraseFile string) {
	g.passphrase, g.passphraseFile = passphrase, passphraseFile
}

// Init verifies availability of gpg & presence of keys
func (g *GoSigner) Init() error {
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
func (g *GoSigner) DetachedSign(source string, destination string) error {
	fmt.Printf("Signing file '%s' with gpg, please enter your passphrase when prompted:\n", filepath.Base(source))

	return nil
}

// ClearSign clear-signs the file
func (g *GoSigner) ClearSign(source string, destination string) error {
	fmt.Printf("Clearsigning file '%s' with gpg, please enter your passphrase when prompted:\n", filepath.Base(source))
	return nil
}

// GoVerifier is implementation of Verifier interface using Go internal OpenPGP library
type GoVerifier struct {
	keyRingFiles []string

	trustedKeyring openpgp.EntityList
}

// InitKeyring verifies that gpg is installed and some keys are trusted
func (g *GoVerifier) InitKeyring() error {
	var err error

	if len(g.keyRingFiles) == 0 {
		g.trustedKeyring, err = loadKeyRing("trustedkeys.gpg", true)
		if err != nil {
			return errors.Wrap(err, "failure loading trustedkeys.gpg keyring")
		}
	} else {
		for _, file := range g.keyRingFiles {
			var keyring openpgp.EntityList

			keyring, err = loadKeyRing(file, false)
			if err != nil {
				return errors.Wrapf(err, "failure loading %s keyring", file)
			}

			g.trustedKeyring = append(g.trustedKeyring, keyring...)
		}
	}

	if len(g.trustedKeyring) == 0 {
		fmt.Printf("\nLooks like your keyring with trusted keys is empty. You might consider importing some keys.\n")
		if len(g.keyRingFiles) == 0 {
			// using default keyring
			fmt.Printf("If you're running Debian or Ubuntu, it's a good idea to import current archive keys by running:\n\n")
			fmt.Printf("  gpg --no-default-keyring --keyring /usr/share/keyrings/debian-archive-keyring.gpg --export | gpg --no-default-keyring --keyring trustedkeys.gpg --import\n")
			fmt.Printf("\n(for Ubuntu, use /usr/share/keyrings/ubuntu-archive-keyring.gpg)\n\n")
		}
	}

	return nil
}

// AddKeyring adds custom keyrings to the list
func (g *GoVerifier) AddKeyring(keyring string) {
	g.keyRingFiles = append(g.keyRingFiles, keyring)
}

func (g *GoVerifier) showImportKeyTip(signers []signatureResult) {
	if len(g.keyRingFiles) == 0 {
		fmt.Printf("\nLooks like some keys are missing in your trusted keyring, you may consider importing them from keyserver:\n\n")

		keys := make([]string, 0)

		for _, signer := range signers {
			if signer.Entity != nil {
				continue
			}
			keys = append(keys, string(KeyFromUint64(signer.IssuerKeyID)))
		}

		fmt.Printf("gpg --no-default-keyring --keyring trustedkeys.gpg --keyserver keys.gnupg.net --recv-keys %s\n\n",
			strings.Join(keys, " "))

		fmt.Printf("Sometimes keys are stored in repository root in file named Release.key, to import such key:\n\n")
		fmt.Printf("wget -O - https://some.repo/repository/Release.key | gpg --no-default-keyring --keyring trustedkeys.gpg --import\n\n")
	}
}

func (g *GoVerifier) printLog(signers []signatureResult) {
	for _, signer := range signers {
		fmt.Printf("openpgp: Signature made %s using %s key ID %s\n",
			signer.CreationTime.Format(time.RFC1123),
			pubkeyAlgorithmName(signer.PubKeyAlgo),
			KeyFromUint64(signer.IssuerKeyID))

		if signer.Entity != nil {
			i := 0
			for name := range signer.Entity.Identities {
				if i == 0 {
					fmt.Printf("openpgp: Good signature from \"%s\"\n", name)
				} else {
					fmt.Printf("                         aka \"%s\"\n", name)
				}
				i++
			}
		} else {
			fmt.Printf("openpgp: Can't check signature: public key not found\n")
		}
	}
}

// VerifyDetachedSignature verifies combination of signature and cleartext using gpgv
func (g *GoVerifier) VerifyDetachedSignature(signature, cleartext io.Reader, showKeyTip bool) error {
	signers, missingKeys, err := checkArmoredDetachedSignature(g.trustedKeyring, cleartext, signature)

	g.printLog(signers)

	if showKeyTip && missingKeys > 0 {
		g.showImportKeyTip(signers)
	}

	if err != nil {
		return errors.Wrap(err, "failed to verify detached signature")
	}

	return nil
}

// IsClearSigned returns true if file contains signature
func (g *GoVerifier) IsClearSigned(clearsigned io.Reader) (bool, error) {
	signedBuffer, err := ioutil.ReadAll(clearsigned)
	if err != nil {
		return false, errors.Wrap(err, "failed to read clearsigned data")
	}

	block, _ := clearsign.Decode(signedBuffer)

	return block != nil, nil
}

// VerifyClearsigned verifies clearsigned file using gpgv
func (g *GoVerifier) VerifyClearsigned(clearsigned io.Reader, showKeyTip bool) (*KeyInfo, error) {
	signedBuffer, err := ioutil.ReadAll(clearsigned)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read clearsigned data")
	}

	block, _ := clearsign.Decode(signedBuffer)
	if block == nil {
		return nil, errors.New("no clearsigned data found")
	}

	signers, missingKeys, err := checkDetachedSignature(g.trustedKeyring, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body)

	g.printLog(signers)

	if showKeyTip && missingKeys > 0 {
		g.showImportKeyTip(signers)
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to verify signature")
	}

	result := &KeyInfo{}

	for _, signer := range signers {
		if signer.Entity != nil {
			result.GoodKeys = append(result.GoodKeys, KeyFromUint64(signer.IssuerKeyID))
		} else {
			result.MissingKeys = append(result.MissingKeys, KeyFromUint64(signer.IssuerKeyID))

		}
	}

	return result, nil
}

// ExtractClearsigned extracts cleartext from clearsigned file WITHOUT signature verification
func (g *GoVerifier) ExtractClearsigned(clearsigned io.Reader) (text *os.File, err error) {
	var signedBuffer []byte
	signedBuffer, err = ioutil.ReadAll(clearsigned)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read clearsigned data")
	}

	block, _ := clearsign.Decode(signedBuffer)
	if block == nil {
		return nil, errors.New("no clearsigned data found")
	}

	text, err = ioutil.TempFile("", "aptly-gpg")
	if err != nil {
		return
	}
	defer os.Remove(text.Name())

	_, err = text.Write(block.Bytes)
	if err != nil {
		return
	}

	_, err = text.Seek(0, 0)

	return
}

var gnupgHome string

func loadKeyRing(name string, ignoreMissing bool) (openpgp.EntityList, error) {
	// if path doesn't contain slashes, treat it as relative to GnuPG home directory
	if !strings.Contains(name, "/") {
		name = filepath.Join(gnupgHome, name)
	}

	f, err := os.Open(name)
	if err != nil {
		if ignoreMissing && os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}
	defer f.Close()

	return openpgp.ReadKeyRing(f)
}

func init() {
	gnupgHome = os.Getenv("GNUPGHOME")
	if gnupgHome == "" {
		// use default location
		gnupgHome = filepath.Join(os.Getenv("HOME"), ".gnupg")
	}
}
