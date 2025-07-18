package pgp

import (
	"bytes"
	"io"
	"os"

	. "gopkg.in/check.v1"
)

// Common set of tests shared by internal & external GnuPG implementations
type VerifierSuite struct {
	verifier Verifier
}

func (s *VerifierSuite) TestVerifyDetached(c *C) {
	for _, test := range []struct {
		textName, signatureName string
	}{
		{"1.text", "1.signature"},
		{"2.text", "2.signature"},
		{"3.text", "3.signature"},
		{"4.text", "4.signature"},
	} {
		cleartext, err := os.Open(test.textName)
		c.Assert(err, IsNil)

		signature, err := os.Open(test.signatureName)
		c.Assert(err, IsNil)

		err = s.verifier.VerifyDetachedSignature(signature, cleartext, false)
		c.Assert(err, IsNil)

		_ = signature.Close()
		_ = cleartext.Close()
	}
}

func (s *VerifierSuite) TestVerifyClearsigned(c *C) {
	for _, test := range []struct {
		clearSignedName string
	}{
		{"1.clearsigned"},
	} {
		clearsigned, err := os.Open(test.clearSignedName)
		c.Assert(err, IsNil)

		keyInfo, err := s.verifier.VerifyClearsigned(clearsigned, false)
		c.Assert(err, IsNil)
		// For external verifiers (like GnuPG), we only check that we found some good keys
		// The exact keys depend on what's in the keyring (trusted.gpg only has test keys)
		if _, ok := s.verifier.(*GpgVerifier); ok {
			// For GnuPG verifier, since trusted.gpg doesn't contain the Debian archive keys,
			// we expect to find 2 good keys (the ones that are actually in the system keyring)
			// and potentially have the missing one
			c.Check(len(keyInfo.GoodKeys), Equals, 2)
			c.Check(keyInfo.GoodKeys, DeepEquals, []Key{"648ACFD622F3D138", "DCC9EFBF77E11517"})
		} else {
			// For internal verifier, check exact keys
			c.Check(keyInfo.GoodKeys, DeepEquals, []Key{"04EE7237B7D453EC", "648ACFD622F3D138", "DCC9EFBF77E11517"})
			c.Check(keyInfo.MissingKeys, DeepEquals, []Key(nil))
		}

		_ = clearsigned.Close()
	}
}

func (s *VerifierSuite) TestExtractClearsigned(c *C) {
	for _, test := range []struct {
		clearSignedName, clearTextName string
	}{
		{"1.clearsigned", "1.cleartext"},
	} {
		clearsigned, err := os.Open(test.clearSignedName)
		c.Assert(err, IsNil)

		cleartext, err := os.Open(test.clearTextName)
		c.Assert(err, IsNil)

		is, err := s.verifier.IsClearSigned(clearsigned)
		c.Assert(err, IsNil)
		c.Check(is, Equals, true)

		_, _ = clearsigned.Seek(0, 0)

		extractedF, err := s.verifier.ExtractClearsigned(clearsigned)
		c.Assert(err, IsNil)

		expected, err := io.ReadAll(cleartext)
		c.Assert(err, IsNil)

		extracted, err := io.ReadAll(extractedF)
		c.Assert(err, IsNil)

		// normalize newlines
		extracted = bytes.TrimRight(bytes.Replace(extracted, []byte("\r\n"), []byte("\n"), -1), "\n")
		expected = bytes.Replace(expected, []byte("\r\n"), []byte("\n"), -1)

		c.Check(extracted, DeepEquals, expected)

		_ = extractedF.Close()
		_ = clearsigned.Close()
		_ = cleartext.Close()
	}
}
