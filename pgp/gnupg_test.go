package pgp

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	. "gopkg.in/check.v1"
)

type GnupgSuite struct {
	bins string
}

var _ = Suite(&GnupgSuite{})

func (s *GnupgSuite) SetUpSuite(c *C) {
	_, _File, _, _ := runtime.Caller(0)
	s.bins = filepath.Join(filepath.Dir(_File), "test-bins")
}

// If gpg == gpg1 = pick gpg
func (s *GnupgSuite) TestGPG1(c *C) {
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", filepath.Join(s.bins, "gpg1"))
	defer func() { _ = os.Setenv("PATH", origPath) }()

	signer := NewGpgSigner(GPG1Finder())
	c.Assert(signer.gpg, Equals, "gpg")
}

// gpg(2) + gpg1 installed = pick gpg1
func (s *GnupgSuite) TestGPG1Not2(c *C) {
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", filepath.Join(s.bins, "gpg2-and-1"))
	defer func() { _ = os.Setenv("PATH", origPath) }()

	signer := NewGpgSigner(GPG1Finder())
	c.Assert(signer.gpg, Equals, "gpg1")
}

// If gpg == gpg2 and no gpg1 is available = error
func (s *GnupgSuite) TestGPGNothing(c *C) {
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", filepath.Join(s.bins, "gpg2-only"))
	defer func() { _ = os.Setenv("PATH", origPath) }()

	c.Assert(func() { NewGpgSigner(GPG1Finder()) }, PanicMatches, `Couldn't find a suitable gpg executable.+`)
}

// If gpgv == gpgv1 = pick gpgv
func (s *GnupgSuite) TestGPGV1(c *C) {
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", filepath.Join(s.bins, "gpgv1")+":"+filepath.Join(s.bins, "gpg1"))
	defer func() { _ = os.Setenv("PATH", origPath) }()

	verifier := NewGpgVerifier(GPG1Finder())
	c.Assert(verifier.gpgv, Equals, "gpgv")
}

// gpgv(2) + gpgv1 installed = pick gpgv1
func (s *GnupgSuite) TestGPGV1Not2(c *C) {
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", filepath.Join(s.bins, "gpgv2-and-1")+":"+filepath.Join(s.bins, "gpg1"))
	defer func() { _ = os.Setenv("PATH", origPath) }()

	verifier := NewGpgVerifier(GPG1Finder())
	c.Assert(verifier.gpgv, Equals, "gpgv1")
}

// If gpgv == gpgv2 and no gpgv1 is available = error
func (s *GnupgSuite) TestGPGVNothing(c *C) {
	origPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", filepath.Join(s.bins, "gpgv2-only")+":"+filepath.Join(s.bins, "gpg1"))
	defer func() { _ = os.Setenv("PATH", origPath) }()

	c.Assert(func() { NewGpgVerifier(GPG1Finder()) }, PanicMatches, `Couldn't find a suitable gpg executable.+`)
}

type Gnupg1VerifierSuite struct {
	VerifierSuite
}

var _ = Suite(&Gnupg1VerifierSuite{})

func (s *Gnupg1VerifierSuite) SetUpTest(c *C) {
	finder := GPG1Finder()
	_, _, err := finder.FindGPG()
	if err != nil {
		c.Skip(err.Error())
	}

	s.verifier = NewGpgVerifier(finder)
	s.verifier.AddKeyring("./trusted.gpg")
	s.multisigOneVerifier = NewGpgVerifier(finder)
	s.multisigOneVerifier.AddKeyring("./multisig-one.gpg")
	s.multisigBothVerifier = NewGpgVerifier(finder)
	s.multisigBothVerifier.AddKeyring("./multisig-both.gpg")

	c.Assert(s.verifier.InitKeyring(false), IsNil)
	c.Assert(s.multisigOneVerifier.InitKeyring(false), IsNil)
	c.Assert(s.multisigBothVerifier.InitKeyring(false), IsNil)
}

type Gnupg1SignerSuite struct {
	SignerSuite
}

var _ = Suite(&Gnupg1SignerSuite{})

func (s *Gnupg1SignerSuite) SetUpTest(c *C) {
	finder := GPG1Finder()
	_, _, err := finder.FindGPG()
	if err != nil {
		c.Skip(err.Error())
	}

	s.keyringNoPassphrase = [2]string{"../system/files/aptly.pub", "../system/files/aptly.sec"}
	s.keyringPassphrase = [2]string{"../system/files/aptly_passphrase.pub", "../system/files/aptly_passphrase.sec"}
	s.passphraseKey = "F30E8CB9CDDE2AF8"
	s.noPassphraseKey = "21DBB89C16DB3E6D"

	s.signer = NewGpgSigner(finder)
	s.signer.SetBatch(true)

	s.verifier = &GoVerifier{}
	s.verifier.AddKeyring("../system/files/aptly.pub")
	s.verifier.AddKeyring("../system/files/aptly_passphrase.pub")

	c.Assert(s.verifier.InitKeyring(false), IsNil)

	s.SignerSuite.SetUpTest(c)
}

type Gnupg2VerifierSuite struct {
	VerifierSuite
}

var _ = Suite(&Gnupg2VerifierSuite{})

func (s *Gnupg2VerifierSuite) SetUpTest(c *C) {
	finder := GPG2Finder()
	_, _, err := finder.FindGPG()
	if err != nil {
		c.Skip(err.Error())
	}

	s.verifier = NewGpgVerifier(finder)
	s.verifier.AddKeyring("./trusted.gpg")
	s.multisigOneVerifier = NewGpgVerifier(finder)
	s.multisigOneVerifier.AddKeyring("./multisig-one.gpg")
	s.multisigBothVerifier = NewGpgVerifier(finder)
	s.multisigBothVerifier.AddKeyring("./multisig-both.gpg")

	c.Assert(s.verifier.InitKeyring(false), IsNil)
	c.Assert(s.multisigOneVerifier.InitKeyring(false), IsNil)
	c.Assert(s.multisigBothVerifier.InitKeyring(false), IsNil)
}

// The status fixtures were generated with GnuPG 2.4 at fixed times so the
// expired key and subkey remain expired without making the other keys age out:
//
//	export GNUPGHOME=$(mktemp -d); chmod 700 "$GNUPGHOME"
//	gpg --batch --pinentry-mode loopback --passphrase '' --faked-system-time 20250101T000000 --quick-generate-key 'Aptly status good fixture <good@fixture.invalid>' rsa2048 sign 0
//	gpg --batch --pinentry-mode loopback --passphrase '' --faked-system-time 20250101T000000 --quick-generate-key 'Aptly status expired fixture <expired@fixture.invalid>' rsa2048 sign seconds=5
//	gpg --batch --pinentry-mode loopback --passphrase '' --faked-system-time 20250101T000000 --quick-generate-key 'Aptly status unknown fixture <unknown-status@fixture.invalid>' rsa2048 sign 0
//	gpg --batch --pinentry-mode loopback --passphrase '' --faked-system-time 20250101T000000 --quick-generate-key 'Aptly status revoked fixture <revoked@fixture.invalid>' rsa2048 sign 0
//	gpg --batch --pinentry-mode loopback --passphrase '' --faked-system-time 20250101T000000 --quick-generate-key 'Aptly unused expired subkey fixture <subkey@fixture.invalid>' rsa2048 cert 0
//	good=A35DB05543372EE999F6EF74EBC0B25127B9BF5A expired=DA1728275CB25D14DC8B6DC97D0DD4A295844522 unknown=A068D4E79AD454D37C0888F5B316843CAB67504E
//	revoked=CD11F407D4CEB092E66A83EC402BF908BCB202A9 primary=E1745C372D24AC29BD57FAB094ABAB0251F5E2FC active=724D68416C95AA282D8280DF22EE1E88523FD531
//	gpg --batch --pinentry-mode loopback --passphrase '' --faked-system-time 20250101T000000 --quick-add-key "$primary" rsa2048 sign 0
//	gpg --batch --pinentry-mode loopback --passphrase '' --faked-system-time 20250101T000000 --quick-add-key "$primary" rsa2048 sign seconds=5
//	gpg --batch --yes --faked-system-time 20250101T000002 --local-user "$good!" --local-user "$expired!" --local-user "$unknown!" --detach-sign --output status-expired-held.signature multisig.text
//	gpg --batch --yes --export "$good" "$expired" > status-expired-held.gpg
//	gpg --batch --yes --faked-system-time 20250101T000002 --local-user "$good!" --local-user "$revoked!" --local-user "$unknown!" --detach-sign --output status-revoked-held.signature multisig.text
//	sed 's/^://' "$GNUPGHOME/openpgp-revocs.d/$revoked.rev" | gpg --batch --yes --import
//	gpg --batch --yes --export "$good" "$revoked" > status-revoked-held.gpg
//	gpg --batch --yes --faked-system-time 20260902T143800 --local-user "$active!" --local-user "$unknown!" --detach-sign --output status-unused-expired-subkey.signature multisig.text
//	gpg --batch --yes --export "$primary" > status-unused-expired-subkey.gpg

func verifyGpgStatusFixture(c *C, finder GPGFinder, keyring, signature string) error {
	verifier := NewGpgVerifier(finder)
	verifier.AddKeyring(keyring)
	c.Assert(verifier.InitKeyring(false), IsNil)

	cleartext, err := os.Open("multisig.text")
	c.Assert(err, IsNil)
	defer func() { _ = cleartext.Close() }()

	signatureFile, err := os.Open(signature)
	c.Assert(err, IsNil)
	defer func() { _ = signatureFile.Close() }()

	return verifier.VerifyDetachedSignature(signatureFile, cleartext, false)
}

func (s *Gnupg2VerifierSuite) TestGpgVerifierRejectsExpiredHeldSignatureAlongsideGood(c *C) {
	err := verifyGpgStatusFixture(c, GPG2Finder(), "./status-expired-held.gpg", "status-expired-held.signature")
	c.Assert(err, NotNil)
}

func (s *Gnupg2VerifierSuite) TestGpgVerifierRejectsRevokedHeldSignatureAlongsideGood(c *C) {
	err := verifyGpgStatusFixture(c, GPG2Finder(), "./status-revoked-held.gpg", "status-revoked-held.signature")
	c.Assert(err, NotNil)
}

func (s *Gnupg2VerifierSuite) TestGpgVerifierIgnoresUnusedExpiredSubkey(c *C) {
	err := verifyGpgStatusFixture(c, GPG2Finder(), "./status-unused-expired-subkey.gpg", "status-unused-expired-subkey.signature")
	c.Assert(err, IsNil)
}

type fixedGPGFinder struct {
	gpgv string
}

func (f fixedGPGFinder) FindGPG() (string, GPGVersion, error) {
	return f.gpgv, GPG22xPlus, nil
}

func (f fixedGPGFinder) FindGPGV() (string, GPGVersion, error) {
	return f.gpgv, GPG22xPlus, nil
}

func (s *GnupgSuite) TestGpgVerifierRejectsUnexplainedFailure(c *C) {
	tests := []struct {
		name        string
		body        string
		wantError   bool
		goodKeys    []Key
		missingKeys []Key
	}{
		{
			name: "unexplained exit",
			body: `#!/bin/sh
printf '%s\n' \
  '[GNUPG:] NEWSIG' \
  '[GNUPG:] GOODSIG 0123456789ABCDEF held' \
  '[GNUPG:] VALIDSIG 00000000000000000123456789ABCDEF 0 0 0 0 0 0 0 00 00000000000000000123456789ABCDEF' >&3
exit 70
`,
			wantError: true,
			goodKeys:  []Key{"0123456789ABCDEF"},
		},
		{
			name: "unknown co-signer",
			body: `#!/bin/sh
printf '%s\n' \
  '[GNUPG:] NEWSIG' \
  '[GNUPG:] GOODSIG 0123456789ABCDEF held' \
  '[GNUPG:] VALIDSIG 00000000000000000123456789ABCDEF 0 0 0 0 0 0 0 00 00000000000000000123456789ABCDEF' \
  '[GNUPG:] NEWSIG' \
  '[GNUPG:] ERRSIG FEDCBA9876543210 1 10 00 0 9 0000000000000000FEDCBA9876543210' \
  '[GNUPG:] NO_PUBKEY FEDCBA9876543210' >&3
exit 2
`,
			goodKeys:    []Key{"0123456789ABCDEF"},
			missingKeys: []Key{"FEDCBA9876543210"},
		},
		{
			name: "non-missing-key ERRSIG",
			body: `#!/bin/sh
printf '%s\n' \
  '[GNUPG:] NEWSIG' \
  '[GNUPG:] GOODSIG 0123456789ABCDEF held' \
  '[GNUPG:] VALIDSIG 00000000000000000123456789ABCDEF 0 0 0 0 0 0 0 00 00000000000000000123456789ABCDEF' \
  '[GNUPG:] NEWSIG' \
  '[GNUPG:] ERRSIG FEDCBA9876543210 1 10 00 0 1 0000000000000000FEDCBA9876543210' >&3
exit 2
`,
			wantError: true,
			goodKeys:  []Key{"0123456789ABCDEF"},
		},
		{
			name: "empty signature block",
			body: `#!/bin/sh
printf '%s\n' \
  '[GNUPG:] NEWSIG' \
  '[GNUPG:] GOODSIG 0123456789ABCDEF held' \
  '[GNUPG:] VALIDSIG 00000000000000000123456789ABCDEF 0 0 0 0 0 0 0 00 00000000000000000123456789ABCDEF' \
  '[GNUPG:] NEWSIG' >&3
exit 2
`,
			wantError: true,
			goodKeys:  []Key{"0123456789ABCDEF"},
		},
		{
			name: "GOODSIG without VALIDSIG",
			body: `#!/bin/sh
printf '%s\n' \
  '[GNUPG:] NEWSIG' \
  '[GNUPG:] GOODSIG 0123456789ABCDEF held' >&3
exit 2
`,
			wantError: true,
			goodKeys:  []Key{"0123456789ABCDEF"},
		},
		{
			name: "signal death",
			body: `#!/bin/sh
printf '%s\n' \
  '[GNUPG:] NEWSIG' \
  '[GNUPG:] GOODSIG 0123456789ABCDEF held' \
  '[GNUPG:] VALIDSIG 00000000000000000123456789ABCDEF 0 0 0 0 0 0 0 00 00000000000000000123456789ABCDEF' >&3
kill -TERM "$$"
`,
			wantError: true,
			goodKeys:  []Key{"0123456789ABCDEF"},
		},
		{
			name: "exit zero preserves upstream behavior",
			body: `#!/bin/sh
printf '%s\n' \
  '[GNUPG:] NEWSIG' \
  '[GNUPG:] BADSIG 0123456789ABCDEF held' >&3
exit 0
`,
		},
	}

	for _, test := range tests {
		directory := c.MkDir()
		gpgv := filepath.Join(directory, "gpgv")
		c.Assert(os.WriteFile(gpgv, []byte(test.body), 0o700), IsNil)

		verifier := NewGpgVerifier(fixedGPGFinder{gpgv: gpgv})
		keyInfo, err := verifier.runGpgv(nil, "detached signature", false)
		if test.wantError {
			c.Check(err, NotNil, Commentf(test.name))
		} else {
			c.Check(err, IsNil, Commentf(test.name))
		}
		c.Check(keyInfo.GoodKeys, DeepEquals, test.goodKeys, Commentf(test.name))
		c.Check(keyInfo.MissingKeys, DeepEquals, test.missingKeys, Commentf(test.name))
	}
}

type Gnupg2SignerSuite struct {
	SignerSuite
}

var _ = Suite(&Gnupg2SignerSuite{})

func (s *Gnupg2SignerSuite) SetUpTest(c *C) {
	finder := GPG2Finder()
	gpg, ver, err := finder.FindGPG()
	if err != nil {
		c.Skip(err.Error())
	}
	if ver == GPG21x {
		c.Skip("skipping sign test on GnuPG 2.1.x, due to loopback pinentry mode troubles")
	}

	// import private keys into gpg2, they're stored outside of keyring files
	for _, item := range []struct {
		suffix string
		key    string
	}{
		{"", "751DF85C2B220D45"},
		{"_passphrase", "6656CD181E92D2D5"},
	} {
		if _, err := exec.Command(gpg, "--list-secret-keys", item.key).CombinedOutput(); err == nil {
			// key already exists
			continue
		}

		args := []string{"--import", "--no-default-keyring", "--batch"}

		if item.suffix == "_passprhase" {
			args = append(args, "--passphrase", "verysecret", "--no-tty")
			if ver >= GPG21x {
				args = append(args, "--pinentry-mode", "loopback")
			}
		}
		args = append(args, "../system/files/aptly2"+item.suffix+".sec.armor")

		output, err := exec.Command(gpg, args...).CombinedOutput()
		c.Log(string(output))
		c.Check(err, IsNil)
	}

	// import public keys into gpg2
	// we can't use pre-built keyrings as gpg 2.0.x and 2.1+ have different keyring formats
	for _, suffix := range []string{"", "_passphrase"} {
		output, err := exec.Command(gpg, "--no-default-keyring", "--batch", "--keyring", "../system/files/aptly2"+suffix+".gpg",
			"--import", "../system/files/aptly2"+suffix+".pub.armor").CombinedOutput()
		c.Log(string(output))
		c.Check(err, IsNil)
	}

	s.keyringNoPassphrase = [2]string{"../system/files/aptly2.gpg", ""}
	s.keyringPassphrase = [2]string{"../system/files/aptly2_passphrase.gpg", ""}
	s.noPassphraseKey = "751DF85C2B220D45"
	s.passphraseKey = "6656CD181E92D2D5"

	s.signer = NewGpgSigner(finder)
	s.signer.SetBatch(true)

	s.verifier = &GoVerifier{}
	s.verifier.AddKeyring("../system/files/aptly2_trusted.pub")

	c.Assert(s.verifier.InitKeyring(false), IsNil)

	s.skipDefaultKey = true

	s.SignerSuite.SetUpTest(c)
}

func (s *Gnupg2SignerSuite) TearDownTest(c *C) {
	s.SignerSuite.TearDownTest(c)

	_ = os.Remove("../system/files/aptly2.gpg")
	_ = os.Remove("../system/files/aptly2_passphrase.gpg")
}
