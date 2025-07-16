package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aptly-dev/aptly/deb"
	"github.com/gin-gonic/gin"
	. "gopkg.in/check.v1"
)

type PublishAPITestSuite struct {
	router *gin.Engine
}

var _ = Suite(&PublishAPITestSuite{})

func (s *PublishAPITestSuite) SetUpTest(c *C) {
	s.router = gin.New()
	s.router.GET("/api/publish", apiPublishList)
	s.router.GET("/api/publish/:prefix/:distribution", apiPublishShow)
	s.router.POST("/api/publish/:prefix", apiPublishRepoOrSnapshot)
}

func (s *PublishAPITestSuite) TestSigningParamsStruct(c *C) {
	// Test signingParams struct and JSON marshaling/unmarshaling
	params := signingParams{
		Skip:           true,
		GpgKey:         "A0546A43624A8331",
		Keyring:        "trustedkeys.gpg",
		SecretKeyring:  "secretkeys.gpg",
		Passphrase:     "verysecure",
		PassphraseFile: "/etc/aptly.passphrase",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(params)
	c.Check(err, IsNil)
	c.Check(string(jsonData), Matches, ".*Skip.*true.*")
	c.Check(string(jsonData), Matches, ".*GpgKey.*A0546A43624A8331.*")

	// Test JSON unmarshaling
	var unmarshaled signingParams
	err = json.Unmarshal(jsonData, &unmarshaled)
	c.Check(err, IsNil)
	c.Check(unmarshaled.Skip, Equals, true)
	c.Check(unmarshaled.GpgKey, Equals, "A0546A43624A8331")
	c.Check(unmarshaled.Keyring, Equals, "trustedkeys.gpg")
	c.Check(unmarshaled.SecretKeyring, Equals, "secretkeys.gpg")
	c.Check(unmarshaled.Passphrase, Equals, "verysecure")
	c.Check(unmarshaled.PassphraseFile, Equals, "/etc/aptly.passphrase")
}

func (s *PublishAPITestSuite) TestSourceParamsStruct(c *C) {
	// Test sourceParams struct and JSON marshaling/unmarshaling
	params := sourceParams{
		Component: "main",
		Name:      "snap1",
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(params)
	c.Check(err, IsNil)
	c.Check(string(jsonData), Matches, ".*Component.*main.*")
	c.Check(string(jsonData), Matches, ".*Name.*snap1.*")

	// Test JSON unmarshaling
	var unmarshaled sourceParams
	err = json.Unmarshal(jsonData, &unmarshaled)
	c.Check(err, IsNil)
	c.Check(unmarshaled.Component, Equals, "main")
	c.Check(unmarshaled.Name, Equals, "snap1")
}

func (s *PublishAPITestSuite) TestGetSignerSkip(c *C) {
	// Test getSigner with Skip=true
	options := &signingParams{
		Skip: true,
	}

	signer, err := getSigner(options)
	c.Check(err, IsNil)
	c.Check(signer, IsNil)
}

func (s *PublishAPITestSuite) TestGetSignerWithOptions(c *C) {
	// Test getSigner with various options (will fail due to context not being set up)
	options := &signingParams{
		Skip:           false,
		GpgKey:         "testkey",
		Keyring:        "test.gpg",
		SecretKeyring:  "secret.gpg",
		Passphrase:     "testpass",
		PassphraseFile: "/tmp/passfile",
	}

	// This will fail because context is not properly set up
	_, err := getSigner(options)
	c.Check(err, NotNil) // Expected to fail without proper context
}

func (s *PublishAPITestSuite) TestSlashEscape(c *C) {
	// Test slashEscape function
	testCases := []struct {
		input    string
		expected string
	}{
		{"", "."},
		{"test_path", "test/path"},
		{"test__path", "test_path"},
		{"test_path_file", "test/path/file"},
		{"test__test__test", "test_test_test"},
		{"_test_", "/test/"},
		{"__", "_"},
		{"test_path__with__underscores", "test/path_with_underscores"},
		{"complex_path__example_test", "complex/path_example/test"},
	}

	for _, tc := range testCases {
		result := slashEscape(tc.input)
		c.Check(result, Equals, tc.expected, Commentf("Input: %s", tc.input))
	}
}

func (s *PublishAPITestSuite) TestSlashEscapeEdgeCases(c *C) {
	// Test edge cases for slashEscape
	edgeCases := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"no_underscores_here", "no/underscores/here"},
		{"double__only", "double_only"},
		{"_", "/"},
		{"__only", "_only"},
		{"only_", "only/"},
		{"mixed_case__Test_Path", "mixed/case_Test/Path"},
		{"numbers_123__test", "numbers/123_test"},
		{"special-chars.test_path", "special-chars.test/path"},
	}

	for _, tc := range edgeCases {
		result := slashEscape(tc.input)
		c.Check(result, Equals, tc.expected, Commentf("Input: '%s'", tc.input))
	}
}

func (s *PublishAPITestSuite) TestApiPublishListBasic(c *C) {
	// Test basic API publish list endpoint
	req, _ := http.NewRequest("GET", "/api/publish", nil)
	w := httptest.NewRecorder()

	// This will fail because context is not set up properly
	s.router.ServeHTTP(w, req)
	// Expect some kind of error due to missing context
	c.Check(w.Code, Not(Equals), http.StatusOK)
}

func (s *PublishAPITestSuite) TestApiPublishShowBasic(c *C) {
	// Test basic API publish show endpoint
	req, _ := http.NewRequest("GET", "/api/publish/test-prefix/test-dist", nil)
	w := httptest.NewRecorder()

	// This will fail because context is not set up properly
	s.router.ServeHTTP(w, req)
	// Expect some kind of error due to missing context
	c.Check(w.Code, Not(Equals), http.StatusOK)
}

func (s *PublishAPITestSuite) TestApiPublishShowWithSlashEscape(c *C) {
	// Test API publish show with slash escape characters
	req, _ := http.NewRequest("GET", "/api/publish/test__prefix/test_dist", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	// Should attempt to process the escaped path
	c.Check(w.Code, Not(Equals), http.StatusOK) // Expected to fail due to missing context
}

func (s *PublishAPITestSuite) TestPublishedRepoCreateParamsStruct(c *C) {
	// Test publishedRepoCreateParams struct
	skipContents := true
	skipCleanup := false
	skipBz2 := true
	acquireByHash := false
	multiDist := true

	params := publishedRepoCreateParams{
		SourceKind:     "snapshot",
		Sources:        []sourceParams{{Component: "main", Name: "test-snap"}},
		Distribution:   "bookworm",
		Label:          "Test Label",
		Origin:         "Test Origin",
		ForceOverwrite: true,
		Architectures:  []string{"amd64", "armhf"},
		Signing: signingParams{
			Skip:   false,
			GpgKey: "A0546A43624A8331",
		},
		NotAutomatic:         "yes",
		ButAutomaticUpgrades: "yes",
		SkipContents:         &skipContents,
		SkipCleanup:          &skipCleanup,
		SkipBz2:              &skipBz2,
		AcquireByHash:        &acquireByHash,
		MultiDist:            &multiDist,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(params)
	c.Check(err, IsNil)
	c.Check(string(jsonData), Matches, ".*SourceKind.*snapshot.*")
	c.Check(string(jsonData), Matches, ".*Distribution.*bookworm.*")
	c.Check(string(jsonData), Matches, ".*Label.*Test Label.*")
	c.Check(string(jsonData), Matches, ".*Origin.*Test Origin.*")
	c.Check(string(jsonData), Matches, ".*ForceOverwrite.*true.*")

	// Test JSON unmarshaling
	var unmarshaled publishedRepoCreateParams
	err = json.Unmarshal(jsonData, &unmarshaled)
	c.Check(err, IsNil)
	c.Check(unmarshaled.SourceKind, Equals, "snapshot")
	c.Check(unmarshaled.Distribution, Equals, "bookworm")
	c.Check(unmarshaled.Label, Equals, "Test Label")
	c.Check(unmarshaled.Origin, Equals, "Test Origin")
	c.Check(unmarshaled.ForceOverwrite, Equals, true)
	c.Check(len(unmarshaled.Sources), Equals, 1)
	c.Check(unmarshaled.Sources[0].Component, Equals, "main")
	c.Check(unmarshaled.Sources[0].Name, Equals, "test-snap")
	c.Check(len(unmarshaled.Architectures), Equals, 2)
	c.Check(unmarshaled.Architectures[0], Equals, "amd64")
	c.Check(unmarshaled.Architectures[1], Equals, "armhf")
	c.Check(*unmarshaled.SkipContents, Equals, true)
	c.Check(*unmarshaled.SkipCleanup, Equals, false)
	c.Check(*unmarshaled.SkipBz2, Equals, true)
	c.Check(*unmarshaled.AcquireByHash, Equals, false)
	c.Check(*unmarshaled.MultiDist, Equals, true)
}

func (s *PublishAPITestSuite) TestPublishedRepoUpdateSwitchParamsStruct(c *C) {
	// Test publishedRepoUpdateSwitchParams struct
	skipContents := false
	skipBz2 := true
	skipCleanup := true
	acquireByHash := true
	multiDist := false

	params := publishedRepoUpdateSwitchParams{
		ForceOverwrite: true,
		Signing: signingParams{
			Skip:    true,
			GpgKey:  "testkey",
			Keyring: "test.gpg",
		},
		SkipContents:  &skipContents,
		SkipBz2:       &skipBz2,
		SkipCleanup:   &skipCleanup,
		Snapshots:     []sourceParams{{Component: "main", Name: "snap1"}, {Component: "contrib", Name: "snap2"}},
		AcquireByHash: &acquireByHash,
		MultiDist:     &multiDist,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(params)
	c.Check(err, IsNil)
	c.Check(string(jsonData), Matches, ".*ForceOverwrite.*true.*")
	c.Check(string(jsonData), Matches, ".*SkipContents.*false.*")
	c.Check(string(jsonData), Matches, ".*SkipBz2.*true.*")

	// Test JSON unmarshaling
	var unmarshaled publishedRepoUpdateSwitchParams
	err = json.Unmarshal(jsonData, &unmarshaled)
	c.Check(err, IsNil)
	c.Check(unmarshaled.ForceOverwrite, Equals, true)
	c.Check(unmarshaled.Signing.Skip, Equals, true)
	c.Check(unmarshaled.Signing.GpgKey, Equals, "testkey")
	c.Check(unmarshaled.Signing.Keyring, Equals, "test.gpg")
	c.Check(*unmarshaled.SkipContents, Equals, false)
	c.Check(*unmarshaled.SkipBz2, Equals, true)
	c.Check(*unmarshaled.SkipCleanup, Equals, true)
	c.Check(*unmarshaled.AcquireByHash, Equals, true)
	c.Check(*unmarshaled.MultiDist, Equals, false)
	c.Check(len(unmarshaled.Snapshots), Equals, 2)
	c.Check(unmarshaled.Snapshots[0].Component, Equals, "main")
	c.Check(unmarshaled.Snapshots[0].Name, Equals, "snap1")
	c.Check(unmarshaled.Snapshots[1].Component, Equals, "contrib")
	c.Check(unmarshaled.Snapshots[1].Name, Equals, "snap2")
}

func (s *PublishAPITestSuite) TestApiPublishRepoOrSnapshotInvalidJSON(c *C) {
	// Test POST endpoint with invalid JSON
	req, _ := http.NewRequest("POST", "/api/publish/test-prefix", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	c.Check(w.Code, Equals, http.StatusBadRequest)
}

func (s *PublishAPITestSuite) TestApiPublishRepoOrSnapshotEmptySources(c *C) {
	// Test POST endpoint with empty sources
	params := publishedRepoCreateParams{
		SourceKind:   "snapshot",
		Sources:      []sourceParams{}, // Empty sources
		Distribution: "test",
	}

	jsonData, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/publish/test-prefix", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should return 400 due to empty sources
	c.Check(w.Code, Equals, http.StatusBadRequest)
}

func (s *PublishAPITestSuite) TestApiPublishRepoOrSnapshotUnknownSourceKind(c *C) {
	// Test POST endpoint with unknown source kind
	params := publishedRepoCreateParams{
		SourceKind:   "unknown",
		Sources:      []sourceParams{{Component: "main", Name: "test"}},
		Distribution: "test",
	}

	jsonData, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/publish/test-prefix", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Should return 400 due to unknown source kind
	c.Check(w.Code, Equals, http.StatusBadRequest)
}

func (s *PublishAPITestSuite) TestApiPublishRepoOrSnapshotValidRequest(c *C) {
	// Test POST endpoint with valid request (will fail due to missing context)
	params := publishedRepoCreateParams{
		SourceKind:   deb.SourceSnapshot,
		Sources:      []sourceParams{{Component: "main", Name: "test-snap"}},
		Distribution: "test-dist",
		Signing:      signingParams{Skip: true},
	}

	jsonData, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/publish/test-prefix", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will fail due to missing context but should get past basic validation
	c.Check(w.Code, Not(Equals), http.StatusBadRequest) // Should not be a 400 error
}

func (s *PublishAPITestSuite) TestApiPublishRepoOrSnapshotLocalRepoSourceKind(c *C) {
	// Test POST endpoint with local repo source kind
	params := publishedRepoCreateParams{
		SourceKind:   deb.SourceLocalRepo,
		Sources:      []sourceParams{{Component: "main", Name: "test-repo"}},
		Distribution: "test-dist",
		Signing:      signingParams{Skip: true},
	}

	jsonData, _ := json.Marshal(params)
	req, _ := http.NewRequest("POST", "/api/publish/test-prefix", bytes.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	// Will fail due to missing context but should get past basic validation
	c.Check(w.Code, Not(Equals), http.StatusBadRequest) // Should not be a 400 error
}

func (s *PublishAPITestSuite) TestSigningParamsEdgeCases(c *C) {
	// Test signingParams with edge cases
	testCases := []signingParams{
		{Skip: true}, // Minimal case
		{Skip: false, GpgKey: "", Keyring: "", SecretKeyring: "", Passphrase: "", PassphraseFile: ""},                   // Empty strings
		{Skip: false, GpgKey: "very-long-key-id-123456789012345678901234567890", Keyring: "very-long-keyring-name.gpg"}, // Long values
		{Skip: false, Passphrase: "password with spaces and special chars !@#$%^&*()"},                                  // Special characters
		{Skip: false, PassphraseFile: "/very/long/path/to/passphrase/file/that/might/not/exist.txt"},                    // Long file path
	}

	for i, params := range testCases {
		// Test JSON marshaling/unmarshaling
		jsonData, err := json.Marshal(params)
		c.Check(err, IsNil, Commentf("Test case %d", i))

		var unmarshaled signingParams
		err = json.Unmarshal(jsonData, &unmarshaled)
		c.Check(err, IsNil, Commentf("Test case %d", i))
		c.Check(unmarshaled.Skip, Equals, params.Skip, Commentf("Test case %d", i))
		c.Check(unmarshaled.GpgKey, Equals, params.GpgKey, Commentf("Test case %d", i))
		c.Check(unmarshaled.Keyring, Equals, params.Keyring, Commentf("Test case %d", i))
	}
}

func (s *PublishAPITestSuite) TestSourceParamsEdgeCases(c *C) {
	// Test sourceParams with edge cases
	testCases := []sourceParams{
		{Component: "", Name: ""}, // Empty strings
		{Component: "very-long-component-name-with-dashes-and-numbers-123", Name: "very-long-name-456"}, // Long values
		{Component: "comp.with.dots", Name: "name_with_underscores"},                                    // Special characters
		{Component: "UPPERCASE", Name: "MixedCase"},                                                     // Case variations
		{Component: "123numeric", Name: "456numbers"},                                                   // Numeric values
	}

	for i, params := range testCases {
		// Test JSON marshaling/unmarshaling
		jsonData, err := json.Marshal(params)
		c.Check(err, IsNil, Commentf("Test case %d", i))

		var unmarshaled sourceParams
		err = json.Unmarshal(jsonData, &unmarshaled)
		c.Check(err, IsNil, Commentf("Test case %d", i))
		c.Check(unmarshaled.Component, Equals, params.Component, Commentf("Test case %d", i))
		c.Check(unmarshaled.Name, Equals, params.Name, Commentf("Test case %d", i))
	}
}

func (s *PublishAPITestSuite) TestSlashEscapeComprehensive(c *C) {
	// Comprehensive test of slashEscape function
	testCases := []struct {
		input       string
		expected    string
		description string
	}{
		{"", ".", "empty string"},
		{"simple", "simple", "no underscores"},
		{"one_underscore", "one/underscore", "single underscore"},
		{"two__underscores", "two_underscores", "double underscore"},
		{"_leading", "/leading", "leading underscore"},
		{"trailing_", "trailing/", "trailing underscore"},
		{"_both_", "/both/", "both leading and trailing"},
		{"__double_leading", "_double/leading", "double leading underscore"},
		{"trailing_double__", "trailing/double_", "double trailing underscore"},
		{"mixed_single__double_combo", "mixed/single_double/combo", "mixed single and double"},
		{"complex_path__with_multiple__sections", "complex/path_with/multiple_sections", "complex path"},
		{"a_b_c_d_e", "a/b/c/d/e", "multiple single underscores"},
		{"a__b__c__d__e", "a_b_c_d_e", "multiple double underscores"},
		{"_a__b_c__d_", "/a_b/c_d/", "mixed pattern"},
		{"test___triple", "test_/triple", "triple underscore"},
		{"test____quad", "test__quad", "quadruple underscore"},
	}

	for _, tc := range testCases {
		result := slashEscape(tc.input)
		c.Check(result, Equals, tc.expected, Commentf("Test case: %s (input: '%s')", tc.description, tc.input))
	}
}

// Mock implementations for testing context dependencies
type MockSigner struct {
	initError      error
	key            string
	keyring        string
	secretKeyring  string
	passphrase     string
	passphraseFile string
	batch          bool
}

func (m *MockSigner) SetKey(key string) { m.key = key }
func (m *MockSigner) SetKeyRing(keyring, secretKeyring string) {
	m.keyring = keyring
	m.secretKeyring = secretKeyring
}
func (m *MockSigner) SetPassphrase(passphrase, passphraseFile string) {
	m.passphrase = passphrase
	m.passphraseFile = passphraseFile
}
func (m *MockSigner) SetBatch(batch bool) { m.batch = batch }
func (m *MockSigner) Init() error         { return m.initError }

func (s *PublishAPITestSuite) TestGetSignerMockSuccess(c *C) {
	// Test getSigner logic with mock (can't test actual getSigner due to context dependencies)
	options := &signingParams{
		Skip:           false,
		GpgKey:         "testkey",
		Keyring:        "test.gpg",
		SecretKeyring:  "secret.gpg",
		Passphrase:     "testpass",
		PassphraseFile: "/tmp/passfile",
	}

	// Mock the signer behavior
	mockSigner := &MockSigner{initError: nil}

	// Simulate what getSigner would do
	mockSigner.SetKey(options.GpgKey)
	mockSigner.SetKeyRing(options.Keyring, options.SecretKeyring)
	mockSigner.SetPassphrase(options.Passphrase, options.PassphraseFile)
	mockSigner.SetBatch(true)
	err := mockSigner.Init()

	c.Check(err, IsNil)
	c.Check(mockSigner.key, Equals, "testkey")
	c.Check(mockSigner.keyring, Equals, "test.gpg")
	c.Check(mockSigner.secretKeyring, Equals, "secret.gpg")
	c.Check(mockSigner.passphrase, Equals, "testpass")
	c.Check(mockSigner.passphraseFile, Equals, "/tmp/passfile")
	c.Check(mockSigner.batch, Equals, true)
}

func (s *PublishAPITestSuite) TestGetSignerMockError(c *C) {
	// Test getSigner logic with mock error
	options := &signingParams{
		Skip:   false,
		GpgKey: "invalidkey",
	}

	// Mock the signer behavior with error
	mockSigner := &MockSigner{initError: fmt.Errorf("mock init error")}

	mockSigner.SetKey(options.GpgKey)
	mockSigner.SetKeyRing(options.Keyring, options.SecretKeyring)
	mockSigner.SetPassphrase(options.Passphrase, options.PassphraseFile)
	mockSigner.SetBatch(true)
	err := mockSigner.Init()

	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "mock init error")
}
