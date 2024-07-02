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
	s.keyringNoPassphrase = [2]string{"keyrings/aptly.pub", "keyrings/aptly.sec"}
	s.keyringPassphrase = [2]string{"keyrings/aptly_passphrase.pub", "keyrings/aptly_passphrase.sec"}
	s.passphraseKey = "F30E8CB9CDDE2AF8"
	s.noPassphraseKey = "21DBB89C16DB3E6D"

	s.signer = &GoSigner{}
	s.signer.SetBatch(true)

	s.verifier = &GoVerifier{}
	s.verifier.AddKeyring("./keyrings/aptly.pub")
	s.verifier.AddKeyring("./keyrings/aptly_passphrase.pub")

	c.Assert(s.verifier.InitKeyring(false), IsNil)

	s.SignerSuite.SetUpTest(c)
}
