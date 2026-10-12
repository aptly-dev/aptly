package pgp

import (
	"bytes"
	"io"
	"os"

	. "gopkg.in/check.v1"
)

// Common set of tests shared by internal & external GnuPG implementations
type VerifierSuite struct {
	verifier             Verifier
	multisigOneVerifier  Verifier
	multisigBothVerifier Verifier
}

// The partial-keyring fixtures were generated with GnuPG 2.4:
//
//	gpg --batch --pinentry-mode loopback --passphrase '' --quick-generate-key 'Aptly multisign known fixture <known@fixture.invalid>' rsa2048 sign 0
//	gpg --batch --pinentry-mode loopback --passphrase '' --quick-generate-key 'Aptly multisign unknown fixture <unknown@fixture.invalid>' rsa2048 sign 0
//	gpg --export 2B5E1D91576341AF44C600E130AD2886ED58775C 32F003A216681D4CC810FA2198B5301EB1FA1317 > multisig-both.gpg
//	gpg --export 2B5E1D91576341AF44C600E130AD2886ED58775C > multisig-one.gpg
//	gpg --local-user 2B5E1D91576341AF44C600E130AD2886ED58775C --local-user 32F003A216681D4CC810FA2198B5301EB1FA1317 --detach-sign --output multisig.signature multisig.text
//	gpg --local-user 2B5E1D91576341AF44C600E130AD2886ED58775C --local-user 32F003A216681D4CC810FA2198B5301EB1FA1317 --clearsign --output multisig.clearsigned multisig.text
//	gpg --local-user 2B5E1D91576341AF44C600E130AD2886ED58775C --detach-sign --output multisig-known-good.signature multisig.text
//	gpg --local-user 32F003A216681D4CC810FA2198B5301EB1FA1317 --detach-sign --output multisig-known-bad.signature multisig-bad.text
//
// multisig-bad.text was made by changing multisig.text after signing it.

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
	clearsigned, err := os.Open("multisig.clearsigned")
	c.Assert(err, IsNil)
	defer func() { _ = clearsigned.Close() }()

	keyInfo, err := s.multisigBothVerifier.VerifyClearsigned(clearsigned, false)
	c.Assert(err, IsNil)
	c.Check(keyInfo.GoodKeys, DeepEquals, []Key{"30AD2886ED58775C", "98B5301EB1FA1317"})
	c.Check(keyInfo.MissingKeys, DeepEquals, []Key(nil))
}

func (s *VerifierSuite) TestVerifyClearsignedPartialKeyring(c *C) {
	clearsigned, err := os.Open("multisig.clearsigned")
	c.Assert(err, IsNil)
	defer func() { _ = clearsigned.Close() }()

	keyInfo, err := s.multisigOneVerifier.VerifyClearsigned(clearsigned, false)
	c.Assert(err, IsNil)
	c.Check(keyInfo.GoodKeys, DeepEquals, []Key{"30AD2886ED58775C"})
	c.Check(keyInfo.MissingKeys, DeepEquals, []Key{"98B5301EB1FA1317"})
}

func (s *VerifierSuite) TestVerifyDetachedPartialKeyring(c *C) {
	cleartext, err := os.Open("multisig.text")
	c.Assert(err, IsNil)
	defer func() { _ = cleartext.Close() }()

	signature, err := os.Open("multisig.signature")
	c.Assert(err, IsNil)
	defer func() { _ = signature.Close() }()

	err = s.multisigOneVerifier.VerifyDetachedSignature(signature, cleartext, false)
	c.Assert(err, IsNil)
}

func (s *VerifierSuite) TestVerifyPartialKeyringRejectsBadSignature(c *C) {
	cleartext, err := os.Open("multisig-bad.text")
	c.Assert(err, IsNil)
	defer func() { _ = cleartext.Close() }()

	signature, err := os.Open("multisig.signature")
	c.Assert(err, IsNil)
	defer func() { _ = signature.Close() }()

	err = s.multisigOneVerifier.VerifyDetachedSignature(signature, cleartext, false)
	c.Assert(err, NotNil)
}

func (s *VerifierSuite) TestVerifyRejectsBadSignatureAlongsideGood(c *C) {
	cleartext, err := os.Open("multisig.text")
	c.Assert(err, IsNil)
	defer func() { _ = cleartext.Close() }()

	goodSignature, err := os.Open("multisig-known-good.signature")
	c.Assert(err, IsNil)
	defer func() { _ = goodSignature.Close() }()

	badSignature, err := os.Open("multisig-known-bad.signature")
	c.Assert(err, IsNil)
	defer func() { _ = badSignature.Close() }()

	signature := io.MultiReader(goodSignature, badSignature)
	err = s.multisigBothVerifier.VerifyDetachedSignature(signature, cleartext, false)
	c.Assert(err, NotNil)
}

func (s *VerifierSuite) TestVerifyNoKnownSignerStillFails(c *C) {
	cleartext, err := os.Open("multisig.text")
	c.Assert(err, IsNil)
	defer func() { _ = cleartext.Close() }()

	signature, err := os.Open("multisig.signature")
	c.Assert(err, IsNil)
	defer func() { _ = signature.Close() }()

	err = s.verifier.VerifyDetachedSignature(signature, cleartext, false)
	c.Assert(err, NotNil)
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
