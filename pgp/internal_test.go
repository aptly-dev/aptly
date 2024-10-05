package pgp

import (
	. "gopkg.in/check.v1"
)

type GoVerifierSuite struct {
	VerifierSuite
}

var _ = Suite(&GoVerifierSuite{})

func (s *GoVerifierSuite) SetUpTest(c *C) {
	s.verifier = &GoVerifier{}
	s.verifier.AddKeyring("./trusted.gpg")

	c.Assert(s.verifier.InitKeyring(false), IsNil)
}

type GoSignerSuite struct {
	SignerSuite
}

var _ = Suite(&GoSignerSuite{})

func (s *GoSignerSuite) SetUpTest(c *C) {
	s.keyringNoPassphrase = [2]string{"../system/files/aptly.pub", "../system/files/aptly.sec"}
	s.keyringPassphrase = [2]string{"../system/files/aptly_passphrase.pub", "../system/files/aptly_passphrase.sec"}
	s.passphraseKey = "F30E8CB9CDDE2AF8"
	s.noPassphraseKey = "21DBB89C16DB3E6D"

	s.signer = &GoSigner{}
	s.signer.SetBatch(true)

	s.verifier = &GoVerifier{}
	s.verifier.AddKeyring("../system/files/aptly.pub")
	s.verifier.AddKeyring("../system/files/aptly_passphrase.pub")

	c.Assert(s.verifier.InitKeyring(false), IsNil)

	s.SignerSuite.SetUpTest(c)
}
