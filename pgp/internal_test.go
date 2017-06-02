package pgp

import (
	"io/ioutil"
	"os"

	. "gopkg.in/check.v1"
)

type GoVerifierSuite struct {
	verifier Verifier
}

var _ = Suite(&GoVerifierSuite{})

func (s *GoVerifierSuite) SetUpTest(c *C) {
	s.verifier = &GoVerifier{}
	s.verifier.AddKeyring("./trusted.gpg")

	c.Assert(s.verifier.InitKeyring(), IsNil)
}

func (s *GoVerifierSuite) TestVerifyDetached(c *C) {
	for _, test := range []struct {
		textName, signatureName string
	}{
		{"1.text", "1.signature"},
		{"2.text", "2.signature"},
		{"3.text", "3.signature"},
	} {
		cleartext, err := os.Open(test.textName)
		c.Assert(err, IsNil)

		signature, err := os.Open(test.signatureName)
		c.Assert(err, IsNil)

		err = s.verifier.VerifyDetachedSignature(signature, cleartext, false)
		c.Assert(err, IsNil)

		signature.Close()
		cleartext.Close()
	}
}

func (s *GoVerifierSuite) TestVerifyClearsigned(c *C) {
	for _, test := range []struct {
		clearSignedName string
	}{
		{"1.clearsigned"},
	} {
		clearsigned, err := os.Open(test.clearSignedName)
		c.Assert(err, IsNil)

		keyInfo, err := s.verifier.VerifyClearsigned(clearsigned, false)
		c.Assert(err, IsNil)
		c.Check(keyInfo.GoodKeys, DeepEquals, []Key{"8B48AD6246925553", "7638D0442B90D010"})
		c.Check(keyInfo.MissingKeys, DeepEquals, []Key(nil))

		clearsigned.Close()
	}
}

func (s *GoVerifierSuite) TestExtractClearsigned(c *C) {
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

		clearsigned.Seek(0, 0)

		extractedF, err := s.verifier.ExtractClearsigned(clearsigned)
		c.Assert(err, IsNil)

		expected, err := ioutil.ReadAll(cleartext)
		c.Assert(err, IsNil)

		extracted, err := ioutil.ReadAll(extractedF)
		c.Assert(err, IsNil)

		c.Check(expected, DeepEquals, extracted)

		extractedF.Close()
		clearsigned.Close()
		cleartext.Close()
	}
}
