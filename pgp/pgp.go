// Package pgp provides interface to signature generation and validation
package pgp

import (
	"fmt"
	"io"
	"os"
)

// Key is key in PGP representation
type Key string

// Matches checks two keys for equality
func (key1 Key) Matches(key2 Key) bool {
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

// KeyFromUint64 converts openpgp uint64 into hex human-readable
func KeyFromUint64(key uint64) Key {
	return Key(fmt.Sprintf("%016X", key))
}

// KeyInfo is response from signature verification
type KeyInfo struct {
	GoodKeys    []Key
	MissingKeys []Key
}

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
	VerifyDetachedSignature(signature, cleartext io.Reader, showKeyTip bool) error
	IsClearSigned(clearsigned io.Reader) (bool, error)
	VerifyClearsigned(clearsigned io.Reader, showKeyTip bool) (*KeyInfo, error)
	ExtractClearsigned(clearsigned io.Reader) (text *os.File, err error)
}
