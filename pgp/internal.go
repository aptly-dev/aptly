package pgp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"

	// TODO: replace crypto/openpgp since it is deprecated
	// https://github.com/golang/go/issues/44226
	"golang.org/x/crypto/openpgp"                       //nolint:staticcheck
	"golang.org/x/crypto/openpgp/clearsign"             //nolint:staticcheck
	openpgp_errors "golang.org/x/crypto/openpgp/errors" //nolint:staticcheck
	"golang.org/x/crypto/openpgp/packet"                //nolint:staticcheck
	"golang.org/x/term"
)

// Test interface
var (
	_ Signer   = &GoSigner{}
	_ Verifier = &GoVerifier{}
)

// Internal errors
var (
	errWrongPassphrase = errors.New("unable to decrypt the key, passphrase is wrong")
)

// GoSigner is implementation of Signer interface using Go internal OpenPGP library
type GoSigner struct {
	keyRef                         string
	keyringFile, secretKeyringFile string
	passphrase, passphraseFile     string
	batch                          bool

	publicKeyring openpgp.EntityList
	secretKeyring openpgp.EntityList
	signer        *openpgp.Entity
	signerConfig  *packet.Config
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
	g.keyringFile, g.secretKeyringFile = keyring, secretKeyring
}

// SetPassphrase sets passphrase params
func (g *GoSigner) SetPassphrase(passphrase, passphraseFile string) {
	g.passphrase, g.passphraseFile = passphrase, passphraseFile
}

// Init verifies availability of gpg & presence of keys
func (g *GoSigner) Init() error {
	g.signerConfig = &packet.Config{
		DefaultCompressionAlgo: packet.CompressionZLIB,
		CompressionConfig: &packet.CompressionConfig{
			Level: 9,
		},
	}

	if g.passphraseFile != "" {
		passF, err := os.Open(g.passphraseFile)
		if err != nil {
			return errors.Wrap(err, "error opening passphrase file")
		}
		defer passF.Close()

		contents, err := io.ReadAll(passF)
		if err != nil {
			return errors.Wrap(err, "error reading passphrase file")
		}

		g.passphrase = strings.TrimSpace(string(contents))
	}

	if g.keyringFile == "" {
		g.keyringFile = "pubring.gpg"
	}

	if g.secretKeyringFile == "" {
		g.secretKeyringFile = "secring.gpg"
	}

	var err error

	g.publicKeyring, err = loadKeyRing(g.keyringFile, false)
	if err != nil {
		return errors.Wrap(err, "error loading public keyring")
	}

	g.secretKeyring, err = loadKeyRing(g.secretKeyringFile, false)
	if err != nil {
		return errors.Wrap(err, "error load secret keyring")
	}

	if g.keyRef == "" {
		// no key reference, pick the first key
		for _, signer := range g.secretKeyring {
			if !validEntity(signer) {
				continue
			}

			g.signer = signer
			break
		}

		if g.signer == nil {
			return fmt.Errorf("looks like there are no keys in gpg, please create one (official manual: http://www.gnupg.org/gph/en/manual.html)")
		}
	} else {
	pickKeyLoop:
		for _, signer := range g.secretKeyring {
			key := KeyFromUint64(signer.PrimaryKey.KeyId)
			if key.Matches(Key(g.keyRef)) {
				g.signer = signer
				break
			}

			if !validEntity(signer) {
				continue
			}

			for name := range signer.Identities {
				if strings.Contains(name, g.keyRef) {
					g.signer = signer
					break pickKeyLoop
				}
			}
		}

		if g.signer == nil {
			return errors.Errorf("couldn't find key for key reference %v", g.keyRef)
		}
	}

	if g.signer.PrivateKey.Encrypted {
		i := 0
		for name := range g.signer.Identities {
			if i == 0 {
				fmt.Printf("openpgp: Passphrase is required to unlock private key \"%s\"\n", name)
			} else {
				fmt.Printf("                         				          aka \"%s\"\n", name)
			}
			i++
		}

		fmt.Printf("openpgp: %s-bit %s key, ID %s, created %s\n",
			keyBits(g.signer.PrimaryKey.PublicKey),
			pubkeyAlgorithmName(g.signer.PrimaryKey.PubKeyAlgo),
			KeyFromUint64(g.signer.PrimaryKey.KeyId),
			g.signer.PrimaryKey.CreationTime.Format("2006-01-02"))

		if g.passphrase == "" {
			if g.batch {
				return errors.New("key is locked with passphrase, but no passphrase was given in batch mode")
			}

			for attempt := 0; attempt < 3; attempt++ {
				fmt.Print("\nEnter passphrase: ")
				var bytePassphrase []byte
				bytePassphrase, err = term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return errors.Wrap(err, "error reading passphare")
				}

				g.passphrase = string(bytePassphrase)

				err = g.decryptKey()
				if err == nil || err != errWrongPassphrase {
					break
				}

				fmt.Print("\nWrong passphrase, please try again.\n")
			}
		} else {
			err = g.decryptKey()
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (g *GoSigner) decryptKey() error {
	err := g.signer.PrivateKey.Decrypt([]byte(g.passphrase))

	if err == nil {
		return nil
	}

	if e, ok := err.(openpgp_errors.StructuralError); ok {
		if string(e) == "private key checksum failure" {
			return errWrongPassphrase
		}
	}

	return errors.Wrap(err, "error unlocking private key")
}

// DetachedSign signs file with detached signature in ASCII format
func (g *GoSigner) DetachedSign(source string, destination string) error {
	fmt.Printf("openpgp: signing file '%s'...\n", filepath.Base(source))

	message, err := os.Open(source)
	if err != nil {
		return errors.Wrap(err, "error opening source file")
	}
	defer message.Close()

	signature, err := os.Create(destination)
	if err != nil {
		return errors.Wrap(err, "error creating signature file")
	}
	defer signature.Close()

	err = openpgp.ArmoredDetachSign(signature, g.signer, message, g.signerConfig)
	if err != nil {
		return errors.Wrap(err, "error creating detached signature")
	}

	return nil
}

// ClearSign clear-signs the file
func (g *GoSigner) ClearSign(source string, destination string) error {
	fmt.Printf("openpgp: clearsigning file '%s'...\n", filepath.Base(source))

	message, err := os.Open(source)
	if err != nil {
		return errors.Wrap(err, "error opening source file")
	}
	defer message.Close()

	clearsigned, err := os.Create(destination)
	if err != nil {
		return errors.Wrap(err, "error creating clearsigned file")
	}
	defer clearsigned.Close()

	stream, err := clearsign.Encode(clearsigned, g.signer.PrivateKey, g.signerConfig)
	if err != nil {
		return errors.Wrap(err, "error initializing clear signer")
	}

	_, err = io.Copy(stream, message)
	if err != nil {
		stream.Close()
		return errors.Wrap(err, "error generating clearsigned signature")
	}

	err = stream.Close()
	if err != nil {
		return errors.Wrap(err, "error generating clearsigned signature")
	}

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

		fmt.Printf("gpg --no-default-keyring --keyring trustedkeys.gpg --keyserver keyserver.ubuntu.com --recv-keys %s\n\n",
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
			names := make([]string, 0, len(signer.Entity.Identities))
			for name := range signer.Entity.Identities {
				names = append(names, name)
			}
			sort.Strings(names)

			for _, name := range names {
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
	var signatureBuf bytes.Buffer

	signers, missingKeys, err := checkArmoredDetachedSignature(g.trustedKeyring, cleartext, io.TeeReader(signature, &signatureBuf))

	if err == io.EOF {
		// most probably not armored signature
		signers, missingKeys, err = checkDetachedSignature(g.trustedKeyring, cleartext, &signatureBuf)
	}

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
	signedBuffer, err := io.ReadAll(clearsigned)
	if err != nil {
		return false, errors.Wrap(err, "failed to read clearsigned data")
	}

	block, _ := clearsign.Decode(signedBuffer)

	return block != nil, nil
}

// VerifyClearsigned verifies clearsigned file using gpgv
func (g *GoVerifier) VerifyClearsigned(clearsigned io.Reader, showKeyTip bool) (*KeyInfo, error) {
	signedBuffer, err := io.ReadAll(clearsigned)
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
	signedBuffer, err = io.ReadAll(clearsigned)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read clearsigned data")
	}

	block, _ := clearsign.Decode(signedBuffer)
	if block == nil {
		return nil, errors.New("no clearsigned data found")
	}

	text, err = os.CreateTemp("", "aptly-gpg")
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
		if os.IsNotExist(err) {
			if !ignoreMissing {
				fmt.Printf("opengpg: failure opening keyring '%s': %s\n", name, err)
			}
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
