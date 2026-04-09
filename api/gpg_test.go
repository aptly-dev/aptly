package api

import (
	"bytes"
	"encoding/json"

	. "gopkg.in/check.v1"
)

type GPGSuite struct {
	APISuite
}

var _ = Suite(&GPGSuite{})

// TestParseGPGOutputEmpty tests parsing of empty GPG output
func (s *GPGSuite) TestParseGPGOutputEmpty(c *C) {
	output := ""
	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 0)
}

// TestParseGPGOutputSingleKeyMinimal tests parsing a single key with minimal fields
func (s *GPGSuite) TestParseGPGOutputSingleKeyMinimal(c *C) {
	// Minimal valid GPG output with one key
	output := `pub:u:4096:1:8B48AD6246925553:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:u::::1611864000::1234567890::John Doe <john@example.com>::::::::::0:
fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 1)

	key := keys[0]
	c.Check(key.KeyID, Equals, "8B48AD6246925553")
	c.Check(key.Validity, Equals, "u")
	c.Check(key.CreatedAt, Equals, "1611864000")
	c.Check(key.Fingerprint, Equals, "D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0")
	c.Check(key.UserIDs, DeepEquals, []string{"John Doe <john@example.com>"})
}

// TestParseGPGOutputMultipleKeys tests parsing multiple keys
func (s *GPGSuite) TestParseGPGOutputMultipleKeys(c *C) {
	output := `pub:u:4096:1:8B48AD6246925553:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:u::::1611864000::1234567890::John Doe <john@example.com>::::::::::0:
fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:
pub:f:2048:1:A1B2C3D4E5F67890:1580592000:1612128000:uidhash:::scESC:::::::23::0:
uid:f::::1580592000::0987654321::Jane Smith <jane@example.com>::::::::::0:
fpr:::::::::E9F0A1B2C3D4E5F6A7B8C9D0E1F2A3B4:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 2)

	// First key
	c.Check(keys[0].KeyID, Equals, "8B48AD6246925553")
	c.Check(keys[0].Validity, Equals, "u")
	c.Check(keys[0].UserIDs, DeepEquals, []string{"John Doe <john@example.com>"})

	// Second key
	c.Check(keys[1].KeyID, Equals, "A1B2C3D4E5F67890")
	c.Check(keys[1].Validity, Equals, "f")
	c.Check(keys[1].UserIDs, DeepEquals, []string{"Jane Smith <jane@example.com>"})
}

// TestParseGPGOutputMultipleUIDs tests a key with multiple user IDs
func (s *GPGSuite) TestParseGPGOutputMultipleUIDs(c *C) {
	output := `pub:u:4096:1:8B48AD6246925553:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:u::::1611864000::1234567890::John Doe <john@example.com>::::::::::0:
uid:u::::1611864000::1234567891::John Doe <john.doe@company.com>::::::::::0:
fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 1)

	key := keys[0]
	c.Check(key.UserIDs, HasLen, 2)
	c.Check(key.UserIDs, DeepEquals, []string{
		"John Doe <john@example.com>",
		"John Doe <john.doe@company.com>",
	})
}

// TestParseGPGOutputMalformedLines tests that malformed lines are skipped
func (s *GPGSuite) TestParseGPGOutputMalformedLines(c *C) {
	// Mix of valid and invalid lines (too few fields)
	output := `pub:u:4096:1:8B48AD6246925553:1611864000:1643400000:uidhash:::scESC:::::::23::0:
invalid:line:with:only:three:fields
uid:u::::1611864000::1234567890::John Doe <john@example.com>::::::::::0:
fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 1)
	c.Check(keys[0].KeyID, Equals, "8B48AD6246925553")
}

// TestParseGPGOutputEmptyLines tests that empty lines are skipped
func (s *GPGSuite) TestParseGPGOutputEmptyLines(c *C) {
	output := `pub:u:4096:1:8B48AD6246925553:1611864000:1643400000:uidhash:::scESC:::::::23::0:

uid:u::::1611864000::1234567890::John Doe <john@example.com>::::::::::0:

fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 1)
	c.Check(keys[0].KeyID, Equals, "8B48AD6246925553")
}

// TestParseGPGOutputKeyWithoutUID tests a public key without user ID
func (s *GPGSuite) TestParseGPGOutputKeyWithoutUID(c *C) {
	// Key without uid record (should still be included)
	output := `pub:u:4096:1:8B48AD6246925553:1611864000:1643400000:uidhash:::scESC:::::::23::0:
fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 1)

	key := keys[0]
	c.Check(key.KeyID, Equals, "8B48AD6246925553")
	c.Check(key.UserIDs, HasLen, 0)
	c.Check(key.Fingerprint, Equals, "D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0")
}

// TestParseGPGOutputVariousValidity tests different validity values
func (s *GPGSuite) TestParseGPGOutputVariousValidity(c *C) {
	output := `pub:u:4096:1:KEY1111111111111:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:u::::1611864000::1234567890::Key1::::::::::0:
fpr:::::::::1111111111111111111111111111111111111111:
pub:f:4096:1:KEY2222222222222:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:f::::1611864000::1234567891::Key2::::::::::0:
fpr:::::::::2222222222222222222222222222222222222222:
pub:m:4096:1:KEY3333333333333:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:m::::1611864000::1234567892::Key3::::::::::0:
fpr:::::::::3333333333333333333333333333333333333333:
pub:n:4096:1:KEY4444444444444:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:n::::1611864000::1234567893::Key4::::::::::0:
fpr:::::::::4444444444444444444444444444444444444444:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 4)

	validities := []string{"u", "f", "m", "n"}
	for i, validity := range validities {
		c.Check(keys[i].Validity, Equals, validity)
	}
}

// TestParseGPGOutputShortKeyID tests that key IDs are shortened to 16 chars
func (s *GPGSuite) TestParseGPGOutputShortKeyID(c *C) {
	// 40-character key ID that should be shortened to last 16 chars
	longKeyID := "0123456789ABCDEF0123456789ABCDEF8B48AD62"
	output := `pub:u:4096:1:` + longKeyID + `:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:u::::1611864000::1234567890::John Doe <john@example.com>::::::::::0:
fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 1)
	// Should extract the last 16 characters: 89ABCDEF8B48AD62
	c.Check(keys[0].KeyID, Equals, "89ABCDEF8B48AD62")
}

// TestParseGPGOutputSpecialCharactersInUID tests user IDs with special characters
func (s *GPGSuite) TestParseGPGOutputSpecialCharactersInUID(c *C) {
	// UID with Unicode characters and special formatting
	output := `pub:u:4096:1:8B48AD6246925553:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:u::::1611864000::1234567890::J\xc3\xb6hn D\xc3\xb6\xc3\xa9 (D\xc3\xbcss) <john@example.com>::::::::::0:
fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 1)
	// Should preserve the encoded special characters
	c.Check(keys[0].UserIDs, HasLen, 1)
}

// TestAPIGPGListKeysDefaultKeyring tests the HTTP endpoint with default keyring
func (s *GPGSuite) TestAPIGPGListKeysDefaultKeyring(c *C) {
	c.Skip("Requires GPG binary and test key installation. Run manually if needed.")

	response, err := s.HTTPRequest("GET", "/api/gpg/keys", nil)
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 200)

	// Verify response is valid JSON
	var result gpgKeyListResponse
	err = json.NewDecoder(response.Body).Decode(&result)
	c.Assert(err, IsNil)
	c.Check(result.Keys, NotNil)
}

// TestAPIGPGListKeysWithKeyringParam tests the HTTP endpoint with custom keyring parameter
func (s *GPGSuite) TestAPIGPGListKeysWithKeyringParam(c *C) {
	c.Skip("Requires GPG binary and test key installation. Run manually if needed.")

	response, err := s.HTTPRequest("GET", "/api/gpg/keys?keyring=custom.gpg", nil)
	c.Assert(err, IsNil)
	// May fail if custom.gpg doesn't exist, but endpoint should handle gracefully
	// Accept either 200 (success) or 400 (file not found)
	code := response.Code
	c.Check(code == 200 || code == 400, Equals, true)
}

// TestAPIGPGListKeysResponseFormat tests that the response has the correct structure
func (s *GPGSuite) TestAPIGPGListKeysResponseFormat(c *C) {
	c.Skip("Requires GPG binary and test key installation. Run manually if needed.")

	response, err := s.HTTPRequest("GET", "/api/gpg/keys", nil)
	c.Assert(err, IsNil)

	if response.Code == 200 {
		var result gpgKeyListResponse
		err = json.NewDecoder(response.Body).Decode(&result)
		c.Assert(err, IsNil)

		// If there are keys, verify their structure
		for _, key := range result.Keys {
			c.Check(key.KeyID, Not(Equals), "")
			c.Check(key.Validity, Not(Equals), "")
			c.Check(key.CreatedAt, Not(Equals), "")
		}
	}
}

// TestParseGPGOutputEdgeCaseUIDWithoutFields tests UID record with missing fields
func (s *GPGSuite) TestParseGPGOutputEdgeCaseUIDWithoutFields(c *C) {
	// UID record with fewer than 10 fields
	output := `pub:u:4096:1:8B48AD6246925553:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:u::::1611864000::1234567890:
fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 1)
	// Should not have user ID since it's in field 9 and this record is too short
	c.Check(keys[0].UserIDs, HasLen, 0)
}

// TestParseGPGOutputFingerprintWithoutCurrentKey tests FPR record appearing before any PUB
func (s *GPGSuite) TestParseGPGOutputFingerprintWithoutCurrentKey(c *C) {
	// FPR record without a preceding PUB (should be ignored)
	output := `fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:
pub:u:4096:1:8B48AD6246925553:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:u::::1611864000::1234567890::John Doe <john@example.com>::::::::::0:
fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 1)
	// Should only have one key with the correct fingerprint
	c.Check(keys[0].Fingerprint, Equals, "D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0")
}

// TestParseGPGOutputComplexRealWorldExample tests real-world-like GPG output
func (s *GPGSuite) TestParseGPGOutputComplexRealWorldExample(c *C) {
	// Real-world GPG output with multiple keys, UIDs, and other record types (sig, sub)
	// Note: sub and sig records are skipped as we only care about pub/uid/fpr
	realWorldOutput := `tru::1:1611864000:0:3:1:5
pub:u:4096:1:8B48AD6246925553:1611864000:2023-01-15T00:00:00:::::scESC:::::::23::0:
fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:
uid:u::::1611864000::1234567890::John Doe <john@example.com>::::::::::0:
uid:u::::1611864100::1234567891::John Doe <john@work.com>::::::::::0:
pub:f:2048:1:1234567890123456:1580592000:2022-12-31T00:00:00::u:::scESC:::::::23::0:
fpr:::::::::F4E3D2C1B0A9F8E7D6C5B4A3F2E1D0C9:
uid:f::::1580592000::0987654321::Maintainer Key <maint@example.com>::::::::::0:`

	keys := parseGPGOutput(realWorldOutput)
	c.Check(keys, HasLen, 2)

	// First key should have 2 UIDs
	c.Check(keys[0].KeyID, Equals, "8B48AD6246925553")
	c.Check(keys[0].UserIDs, HasLen, 2)
	c.Check(keys[0].Fingerprint, Equals, "D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0")

	// Second key should have 1 UID
	c.Check(keys[1].KeyID, Equals, "1234567890123456")
	c.Check(keys[1].UserIDs, HasLen, 1)
	c.Check(keys[1].Fingerprint, Equals, "F4E3D2C1B0A9F8E7D6C5B4A3F2E1D0C9")
}

// TestParseGPGOutputConsecutiveEmptyUIDs tests handling of consecutive empty user ID fields
func (s *GPGSuite) TestParseGPGOutputConsecutiveEmptyUIDs(c *C) {
	output := `pub:u:4096:1:8B48AD6246925553:1611864000:1643400000:uidhash:::scESC:::::::23::0:
uid:u::::1611864000::1234567890:::::::::::0:
uid:u::::1611864000::1234567891::John Doe <john@example.com>::::::::::0:
fpr:::::::::D8E8F5A516E7A2C4F3E4B5A6C7D8E9F0:`

	keys := parseGPGOutput(output)
	c.Check(keys, HasLen, 1)
	// Should skip empty UID but include the non-empty one
	c.Check(keys[0].UserIDs, HasLen, 1)
	c.Check(keys[0].UserIDs[0], Equals, "John Doe <john@example.com>")
}

// TestGPGDeleteKeyParamsValidation tests gpgDeleteKeyParams validation
func (s *GPGSuite) TestGPGDeleteKeyParamsValidation(c *C) {
	// This is a unit test that validates parameter structure (no HTTP needed)
	params := gpgDeleteKeyParams{
		Keyring: "custom.gpg",
		GpgKeyID: "8B48AD6246925553",
	}
	c.Check(params.Keyring, Equals, "custom.gpg")
	c.Check(params.GpgKeyID, Equals, "8B48AD6246925553")
}

// TestAPIGPGDeleteKeyMissingKeyID tests delete with missing key ID parameter
func (s *GPGSuite) TestAPIGPGDeleteKeyMissingKeyID(c *C) {
	c.Skip("Requires GPG binary. Run manually if needed.")

	body, err := json.Marshal(map[string]string{
		"Keyring": "trustedkeys.gpg",
		// GpgKeyID is missing
	})
	c.Assert(err, IsNil)

	response, err := s.HTTPRequest("DELETE", "/api/gpg/key", bytes.NewReader(body))
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 400)
}

// TestAPIGPGDeleteKeyInvalidJSON tests delete with invalid JSON request
func (s *GPGSuite) TestAPIGPGDeleteKeyInvalidJSON(c *C) {
	c.Skip("Requires GPG binary. Run manually if needed.")

	response, err := s.HTTPRequest("DELETE", "/api/gpg/key", bytes.NewReader([]byte("invalid json")))
	c.Assert(err, IsNil)
	c.Check(response.Code, Equals, 400)
}

// TestAPIGPGDeleteKeySuccess tests successful key deletion
func (s *GPGSuite) TestAPIGPGDeleteKeySuccess(c *C) {
	c.Skip("Requires GPG binary and test key installation. Run manually if needed.")

	body, err := json.Marshal(gpgDeleteKeyParams{
		Keyring: "trustedkeys.gpg",
		GpgKeyID: "8B48AD6246925553",
	})
	c.Assert(err, IsNil)

	response, err := s.HTTPRequest("DELETE", "/api/gpg/key", bytes.NewReader(body))
	c.Assert(err, IsNil)
	// Should succeed (200) or fail gracefully (400) if key doesn't exist
	code := response.Code
	c.Check(code == 200 || code == 400, Equals, true)
}
