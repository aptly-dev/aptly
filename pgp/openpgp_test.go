package pgp

import (
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"math/big"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/errors"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	. "gopkg.in/check.v1"
)

type OpenPGPSuite struct{}

var _ = Suite(&OpenPGPSuite{})

func (s *OpenPGPSuite) TestHashForSignatureBinary(c *C) {
	// Test hash creation for binary signature
	hashFunc := crypto.SHA256

	h1, h2, err := hashForSignature(hashFunc, packet.SigTypeBinary)
	c.Check(err, IsNil)
	c.Check(h1, NotNil)
	c.Check(h2, NotNil)

	// For binary signatures, both hashes should be the same instance
	c.Check(h1, Equals, h2)
}

func (s *OpenPGPSuite) TestHashForSignatureText(c *C) {
	// Test hash creation for text signature
	hashFunc := crypto.SHA256

	h1, h2, err := hashForSignature(hashFunc, packet.SigTypeText)
	c.Check(err, IsNil)
	c.Check(h1, NotNil)
	c.Check(h2, NotNil)

	// For text signatures, h2 should be a canonical text hash wrapper
	c.Check(h1, Not(Equals), h2)
}

func (s *OpenPGPSuite) TestHashForSignatureUnsupportedHash(c *C) {
	// Test with unsupported hash algorithm
	hashFunc := crypto.Hash(999) // Invalid hash

	h1, h2, err := hashForSignature(hashFunc, packet.SigTypeBinary)
	c.Check(err, NotNil)
	c.Check(h1, IsNil)
	c.Check(h2, IsNil)
	c.Check(err.Error(), Matches, ".*hash not available.*")
}

func (s *OpenPGPSuite) TestHashForSignatureUnsupportedSigType(c *C) {
	// Test with unsupported signature type
	hashFunc := crypto.SHA256

	h1, h2, err := hashForSignature(hashFunc, packet.SignatureType(255))
	c.Check(err, NotNil)
	c.Check(h1, IsNil)
	c.Check(h2, IsNil)
	c.Check(err.Error(), Matches, ".*unsupported signature type.*")
}

func (s *OpenPGPSuite) TestSignatureResultStruct(c *C) {
	// Test signatureResult struct creation and field access
	now := time.Now()
	keyID := uint64(0x1234567890ABCDEF)

	result := signatureResult{
		CreationTime: now,
		IssuerKeyID:  keyID,
		PubKeyAlgo:   packet.PubKeyAlgoRSA,
		Entity:       nil, // Can be nil for missing keys
	}

	c.Check(result.CreationTime, Equals, now)
	c.Check(result.IssuerKeyID, Equals, keyID)
	c.Check(result.PubKeyAlgo, Equals, packet.PubKeyAlgoRSA)
	c.Check(result.Entity, IsNil)
}

func (s *OpenPGPSuite) TestCheckDetachedSignatureNoSignature(c *C) {
	// Test with empty signature
	keyring := openpgp.EntityList{}
	signed := strings.NewReader("test data")
	signature := strings.NewReader("")

	signers, missingKeys, err := checkDetachedSignature(keyring, signed, signature)
	c.Check(err, Equals, errors.ErrUnknownIssuer)
	c.Check(len(signers), Equals, 0)
	c.Check(missingKeys, Equals, 0)
}

func (s *OpenPGPSuite) TestCheckDetachedSignatureInvalidPacket(c *C) {
	// Test with invalid packet data
	keyring := openpgp.EntityList{}
	signed := strings.NewReader("test data")
	signature := strings.NewReader("invalid packet data")

	signers, missingKeys, err := checkDetachedSignature(keyring, signed, signature)
	c.Check(err, NotNil)
	c.Check(len(signers), Equals, 0)
	c.Check(missingKeys, Equals, 0)
}

func (s *OpenPGPSuite) TestReadArmoredValidBlock(c *C) {
	// Test reading valid armored block
	armoredData := `-----BEGIN PGP SIGNATURE-----

iQEcBAABAgAGBQJeRllaAAoJEDvKaJaAL9sRiUUH/test
-----END PGP SIGNATURE-----`

	reader := strings.NewReader(armoredData)
	body, err := readArmored(reader, "PGP SIGNATURE")
	c.Check(err, IsNil)
	c.Check(body, NotNil)
}

func (s *OpenPGPSuite) TestReadArmoredWrongType(c *C) {
	// Test reading armored block with wrong type
	armoredData := `-----BEGIN PGP MESSAGE-----

test
-----END PGP MESSAGE-----`

	reader := strings.NewReader(armoredData)
	body, err := readArmored(reader, "PGP SIGNATURE")
	c.Check(err, NotNil)
	c.Check(err.Error(), Matches, ".*expected 'PGP SIGNATURE', got: PGP MESSAGE.*")
	c.Check(body, IsNil)
}

func (s *OpenPGPSuite) TestReadArmoredInvalidArmor(c *C) {
	// Test reading invalid armored data
	reader := strings.NewReader("not armored data")
	body, err := readArmored(reader, "PGP SIGNATURE")
	c.Check(err, NotNil)
	c.Check(body, IsNil)
}

func (s *OpenPGPSuite) TestCheckArmoredDetachedSignatureInvalidArmor(c *C) {
	// Test with invalid armored signature
	keyring := openpgp.EntityList{}
	signed := strings.NewReader("test data")
	signature := strings.NewReader("not armored")

	signers, missingKeys, err := checkArmoredDetachedSignature(keyring, signed, signature)
	c.Check(err, NotNil)
	c.Check(len(signers), Equals, 0)
	c.Check(missingKeys, Equals, 0)
}

func (s *OpenPGPSuite) TestPubkeyAlgorithmNameRSA(c *C) {
	// Test RSA algorithm names
	c.Check(pubkeyAlgorithmName(packet.PubKeyAlgoRSA), Equals, "RSA")
	c.Check(pubkeyAlgorithmName(packet.PubKeyAlgoRSAEncryptOnly), Equals, "RSA")
	c.Check(pubkeyAlgorithmName(packet.PubKeyAlgoRSASignOnly), Equals, "RSA")
}

func (s *OpenPGPSuite) TestPubkeyAlgorithmNameOthers(c *C) {
	// Test other algorithm names
	c.Check(pubkeyAlgorithmName(packet.PubKeyAlgoElGamal), Equals, "ElGamal")
	c.Check(pubkeyAlgorithmName(packet.PubKeyAlgoDSA), Equals, "DSA")
	c.Check(pubkeyAlgorithmName(packet.PubKeyAlgoECDH), Equals, "EDCH")
	c.Check(pubkeyAlgorithmName(packet.PubKeyAlgoECDSA), Equals, "ECDSA")
	c.Check(pubkeyAlgorithmName(packet.PubKeyAlgoEdDSA), Equals, "EdDSA")
}

func (s *OpenPGPSuite) TestPubkeyAlgorithmNameUnknown(c *C) {
	// Test unknown algorithm
	c.Check(pubkeyAlgorithmName(packet.PublicKeyAlgorithm(255)), Equals, "unknown")
}

func (s *OpenPGPSuite) TestKeyBitsRSA(c *C) {
	// Test RSA key bit calculation
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	c.Check(err, IsNil)

	bits := keyBits(&rsaKey.PublicKey)
	c.Check(bits, Equals, "2048")
}

func (s *OpenPGPSuite) TestKeyBitsDSA(c *C) {
	// Test DSA key bit calculation
	dsaKey := &dsa.PublicKey{
		Parameters: dsa.Parameters{
			P: big.NewInt(0).SetBit(big.NewInt(0), 1024, 1), // 2^1024
		},
	}

	bits := keyBits(dsaKey)
	c.Check(bits, Equals, "1025") // SetBit creates a number with bit 1024 set
}

func (s *OpenPGPSuite) TestKeyBitsECDSA(c *C) {
	// Test ECDSA key bit calculation
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	c.Check(err, IsNil)

	bits := keyBits(&ecdsaKey.PublicKey)
	c.Check(bits, Equals, "256") // P256 curve
}

func (s *OpenPGPSuite) TestKeyBitsUnknown(c *C) {
	// Test unknown key type
	bits := keyBits("unknown key type")
	c.Check(bits, Equals, "?")
}

func (s *OpenPGPSuite) TestValidEntityNoIdentities(c *C) {
	// Test entity with no identities
	entity := &openpgp.Entity{
		Identities: make(map[string]*openpgp.Identity),
	}

	valid := validEntity(entity)
	c.Check(valid, Equals, false)
}

func (s *OpenPGPSuite) TestValidEntityWithRevocations(c *C) {
	// Test entity with revocations
	entity := &openpgp.Entity{
		Identities: map[string]*openpgp.Identity{
			"test": {
				SelfSignature: &packet.Signature{
					FlagsValid: true,
				},
			},
		},
		Revocations: []*packet.Signature{
			{}, // Has revocation
		},
	}

	valid := validEntity(entity)
	c.Check(valid, Equals, false)
}

func (s *OpenPGPSuite) TestValidEntityWithRevocationReason(c *C) {
	// Test entity with revocation reason
	entity := &openpgp.Entity{
		Identities: map[string]*openpgp.Identity{
			"test": {
				SelfSignature: &packet.Signature{
					RevocationReason: nil,
				},
			},
		},
	}

	valid := validEntity(entity)
	c.Check(valid, Equals, false)
}

func (s *OpenPGPSuite) TestValidEntityInvalidFlags(c *C) {
	// Test entity with invalid flags
	entity := &openpgp.Entity{
		Identities: map[string]*openpgp.Identity{
			"test": {
				SelfSignature: &packet.Signature{
					FlagsValid: false,
				},
			},
		},
	}

	valid := validEntity(entity)
	c.Check(valid, Equals, false)
}

func (s *OpenPGPSuite) TestValidEntityExpired(c *C) {
	// Test entity that has expired
	keyLifetime := uint32(1) // 1 second lifetime
	entity := &openpgp.Entity{
		Identities: map[string]*openpgp.Identity{
			"test": {
				SelfSignature: &packet.Signature{
					FlagsValid:      true,
					CreationTime:    time.Now().Add(-time.Hour), // Created 1 hour ago
					KeyLifetimeSecs: &keyLifetime,
				},
			},
		},
	}

	valid := validEntity(entity)
	c.Check(valid, Equals, false)
}

func (s *OpenPGPSuite) TestValidEntityMultipleIdentitiesPrimary(c *C) {
	// Test entity with multiple identities, one marked as primary
	isPrimary := true
	isNotPrimary := false

	entity := &openpgp.Entity{
		Identities: map[string]*openpgp.Identity{
			"secondary": {
				SelfSignature: &packet.Signature{
					FlagsValid:   true,
					CreationTime: time.Now(),
					IsPrimaryId:  &isNotPrimary,
				},
			},
			"primary": {
				SelfSignature: &packet.Signature{
					FlagsValid:   true,
					CreationTime: time.Now(),
					IsPrimaryId:  &isPrimary,
				},
			},
		},
	}

	valid := validEntity(entity)
	c.Check(valid, Equals, true)
}

func (s *OpenPGPSuite) TestValidEntityValidCase(c *C) {
	// Test valid entity
	entity := &openpgp.Entity{
		Identities: map[string]*openpgp.Identity{
			"test": {
				SelfSignature: &packet.Signature{
					FlagsValid:   true,
					CreationTime: time.Now(),
				},
			},
		},
	}

	valid := validEntity(entity)
	c.Check(valid, Equals, true)
}

func (s *OpenPGPSuite) TestCheckDetachedSignatureEmptyReader(c *C) {
	// Test with empty signed data reader
	keyring := openpgp.EntityList{}
	signed := strings.NewReader("")
	signature := strings.NewReader("")

	signers, missingKeys, err := checkDetachedSignature(keyring, signed, signature)
	c.Check(err, Equals, errors.ErrUnknownIssuer)
	c.Check(len(signers), Equals, 0)
	c.Check(missingKeys, Equals, 0)
}

func (s *OpenPGPSuite) TestCheckDetachedSignatureErrorInCopy(c *C) {
	// Test error handling during copy operation
	keyring := openpgp.EntityList{}
	signed := &errorReader{} // Custom reader that returns error
	signature := strings.NewReader("")

	signers, missingKeys, err := checkDetachedSignature(keyring, signed, signature)
	c.Check(err, NotNil)
	c.Check(len(signers), Equals, 0)
	c.Check(missingKeys, Equals, 0)
}

func (s *OpenPGPSuite) TestReadArmoredEmptyReader(c *C) {
	// Test with empty reader
	reader := strings.NewReader("")
	body, err := readArmored(reader, "PGP SIGNATURE")
	c.Check(err, NotNil)
	c.Check(body, IsNil)
}

func (s *OpenPGPSuite) TestHashForSignatureAllSupportedHashes(c *C) {
	// Test with all commonly supported hash algorithms
	supportedHashes := []crypto.Hash{
		crypto.SHA1,
		crypto.SHA224,
		crypto.SHA256,
		crypto.SHA384,
		crypto.SHA512,
	}

	for _, hashFunc := range supportedHashes {
		if hashFunc.Available() {
			h1, h2, err := hashForSignature(hashFunc, packet.SigTypeBinary)
			c.Check(err, IsNil, Commentf("Failed for hash: %v", hashFunc))
			c.Check(h1, NotNil)
			c.Check(h2, NotNil)
		}
	}
}

func (s *OpenPGPSuite) TestSignatureResultZeroValues(c *C) {
	// Test signatureResult with zero values
	result := signatureResult{}

	c.Check(result.CreationTime.IsZero(), Equals, true)
	c.Check(result.IssuerKeyID, Equals, uint64(0))
	c.Check(result.PubKeyAlgo, Equals, packet.PublicKeyAlgorithm(0))
	c.Check(result.Entity, IsNil)
}

// Mock error reader for testing error conditions
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func (s *OpenPGPSuite) TestArmorDecodeCornerCases(c *C) {
	// Test various armor decode edge cases
	testCases := []struct {
		name      string
		input     string
		expected  string
		shouldErr bool
	}{
		{
			name: "empty armor block",
			input: `-----BEGIN PGP SIGNATURE-----

-----END PGP SIGNATURE-----`,
			expected:  "PGP SIGNATURE",
			shouldErr: false,
		},
		{
			name: "armor with headers",
			input: `-----BEGIN PGP SIGNATURE-----
Version: GnuPG v1

test
-----END PGP SIGNATURE-----`,
			expected:  "PGP SIGNATURE",
			shouldErr: false,
		},
		{
			name: "malformed armor start",
			input: `----BEGIN PGP SIGNATURE-----
test
-----END PGP SIGNATURE-----`,
			expected:  "",
			shouldErr: true,
		},
		{
			name: "malformed armor end",
			input: `-----BEGIN PGP SIGNATURE-----
test
----END PGP SIGNATURE-----`,
			expected:  "",
			shouldErr: true,
		},
	}

	for _, tc := range testCases {
		reader := strings.NewReader(tc.input)
		body, err := readArmored(reader, tc.expected)

		if tc.shouldErr {
			c.Check(err, NotNil, Commentf("Test case: %s", tc.name))
			c.Check(body, IsNil, Commentf("Test case: %s", tc.name))
		} else {
			c.Check(err, IsNil, Commentf("Test case: %s", tc.name))
			c.Check(body, NotNil, Commentf("Test case: %s", tc.name))
		}
	}
}

func (s *OpenPGPSuite) TestKeyBitsEdgeCases(c *C) {
	// Test keyBits function with edge cases
	testCases := []struct {
		name     string
		key      interface{}
		expected string
	}{
		{
			name:     "nil key",
			key:      nil,
			expected: "?",
		},
		{
			name:     "string key",
			key:      "not a key",
			expected: "?",
		},
		{
			name:     "int key",
			key:      123,
			expected: "?",
		},
		{
			name:     "slice key",
			key:      []byte{1, 2, 3},
			expected: "?",
		},
	}

	for _, tc := range testCases {
		result := keyBits(tc.key)
		c.Check(result, Equals, tc.expected, Commentf("Test case: %s", tc.name))
	}
}

func (s *OpenPGPSuite) TestValidEntityEdgeCases(c *C) {
	// Test validEntity with various edge cases

	// Entity with nil self-signature
	entity1 := &openpgp.Entity{
		Identities: map[string]*openpgp.Identity{
			"test": {
				SelfSignature: nil,
			},
		},
	}
	c.Check(validEntity(entity1), Equals, false)

	// Entity with key that never expires (nil KeyLifetimeSecs)
	entity2 := &openpgp.Entity{
		Identities: map[string]*openpgp.Identity{
			"test": {
				SelfSignature: &packet.Signature{
					FlagsValid:      true,
					CreationTime:    time.Now(),
					KeyLifetimeSecs: nil, // Never expires
				},
			},
		},
	}
	c.Check(validEntity(entity2), Equals, true)

	// Entity with key that expires in the future
	futureLifetime := uint32(3600) // 1 hour from creation
	entity3 := &openpgp.Entity{
		Identities: map[string]*openpgp.Identity{
			"test": {
				SelfSignature: &packet.Signature{
					FlagsValid:      true,
					CreationTime:    time.Now(),
					KeyLifetimeSecs: &futureLifetime,
				},
			},
		},
	}
	c.Check(validEntity(entity3), Equals, true)
}
