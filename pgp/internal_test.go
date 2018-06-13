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

	c.Assert(s.verifier.InitKeyring(), IsNil)
}

type GoSignerSuite struct {
	SignerSuite
}

var _ = Suite(&GoSignerSuite{})

func (s *GoSignerSuite) SetUpTest(c *C) {
	s.signer = &GoSigner{}
	s.signer.SetBatch(true)

	s.verifier = &GoVerifier{}
	s.verifier.AddKeyring("./keyrings/aptly.pub")
	s.verifier.AddKeyring("./keyrings/aptly_passphrase.pub")

	c.Assert(s.verifier.InitKeyring(), IsNil)

	s.SignerSuite.SetUpTest(c)
}
