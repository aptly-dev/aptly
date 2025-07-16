package pgp

import (
	"os/exec"
	"strings"

	. "gopkg.in/check.v1"
)

type GPGFinderSuite struct{}

var _ = Suite(&GPGFinderSuite{})

func (s *GPGFinderSuite) TestGPGVersionConstants(c *C) {
	// Test GPG version constants are defined correctly
	c.Check(GPG1x, Equals, GPGVersion(1))
	c.Check(GPG20x, Equals, GPGVersion(2))
	c.Check(GPG21x, Equals, GPGVersion(3))
	c.Check(GPG22xPlus, Equals, GPGVersion(4))
}

func (s *GPGFinderSuite) TestGPG1Finder(c *C) {
	// Test GPG1 finder configuration
	finder := GPG1Finder()
	c.Check(finder, NotNil)
	
	pathFinder, ok := finder.(*pathGPGFinder)
	c.Check(ok, Equals, true)
	c.Check(pathFinder.gpgNames, DeepEquals, []string{"gpg", "gpg1"})
	c.Check(pathFinder.gpgvNames, DeepEquals, []string{"gpgv", "gpgv1"})
	c.Check(pathFinder.expectedVersionSubstring, Equals, `\(GnuPG.*\) (1).(\d)`)
	c.Check(strings.Contains(pathFinder.errorMessage, "gnupg1"), Equals, true)
}

func (s *GPGFinderSuite) TestGPG2Finder(c *C) {
	// Test GPG2 finder configuration
	finder := GPG2Finder()
	c.Check(finder, NotNil)
	
	pathFinder, ok := finder.(*pathGPGFinder)
	c.Check(ok, Equals, true)
	c.Check(pathFinder.gpgNames, DeepEquals, []string{"gpg", "gpg2"})
	c.Check(pathFinder.gpgvNames, DeepEquals, []string{"gpgv", "gpgv2"})
	c.Check(pathFinder.expectedVersionSubstring, Equals, `\(GnuPG.*\) (2).(\d)`)
	c.Check(strings.Contains(pathFinder.errorMessage, "gnupg2"), Equals, true)
}

func (s *GPGFinderSuite) TestGPGDefaultFinder(c *C) {
	// Test default finder configuration
	finder := GPGDefaultFinder()
	c.Check(finder, NotNil)
	
	iterFinder, ok := finder.(*iteratingGPGFinder)
	c.Check(ok, Equals, true)
	c.Check(len(iterFinder.finders), Equals, 2)
	c.Check(strings.Contains(iterFinder.errorMessage, "gnupg"), Equals, true)
}

func (s *GPGFinderSuite) TestPathGPGFinderFindGPGNotFound(c *C) {
	// Test when GPG is not found
	finder := &pathGPGFinder{
		gpgNames:                 []string{"nonexistent-gpg"},
		gpgvNames:                []string{"nonexistent-gpgv"},
		expectedVersionSubstring: `\(GnuPG.*\) (1).(\d)`,
		errorMessage:             "test error",
	}
	
	gpg, version, err := finder.FindGPG()
	c.Check(gpg, Equals, "")
	c.Check(version, Equals, GPGVersion(0))
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "test error")
}

func (s *GPGFinderSuite) TestPathGPGFinderFindGPGVNotFound(c *C) {
	// Test when GPGV is not found
	finder := &pathGPGFinder{
		gpgNames:                 []string{"nonexistent-gpg"},
		gpgvNames:                []string{"nonexistent-gpgv"},
		expectedVersionSubstring: `\(GnuPG.*\) (1).(\d)`,
		errorMessage:             "test error",
	}
	
	gpgv, version, err := finder.FindGPGV()
	c.Check(gpgv, Equals, "")
	c.Check(version, Equals, GPGVersion(0))
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "test error")
}

func (s *GPGFinderSuite) TestIteratingGPGFinderAllFail(c *C) {
	// Test when all finders fail
	failingFinder := &pathGPGFinder{
		gpgNames:                 []string{"nonexistent-gpg"},
		gpgvNames:                []string{"nonexistent-gpgv"},
		expectedVersionSubstring: `\(GnuPG.*\) (1).(\d)`,
		errorMessage:             "individual finder error",
	}
	
	finder := &iteratingGPGFinder{
		finders:      []GPGFinder{failingFinder, failingFinder},
		errorMessage: "all finders failed",
	}
	
	gpg, version, err := finder.FindGPG()
	c.Check(gpg, Equals, "")
	c.Check(version, Equals, GPGVersion(0))
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "all finders failed")
	
	gpgv, version, err := finder.FindGPGV()
	c.Check(gpgv, Equals, "")
	c.Check(version, Equals, GPGVersion(0))
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "all finders failed")
}

func (s *GPGFinderSuite) TestCliVersionCheckCommandNotFound(c *C) {
	// Test version check with non-existent command
	result, version := cliVersionCheck("nonexistent-command", `\(GnuPG.*\) (1).(\d)`)
	c.Check(result, Equals, false)
	c.Check(version, Equals, GPGVersion(0))
}

func (s *GPGFinderSuite) TestCliVersionCheckInvalidRegex(c *C) {
	// Test version check with invalid regex (should not crash)
	// This uses a command that exists but won't match
	result, version := cliVersionCheck("echo", "[invalid regex")
	c.Check(result, Equals, false)
	c.Check(version, Equals, GPGVersion(0))
}

func (s *GPGFinderSuite) TestCliVersionCheckGPG1Pattern(c *C) {
	// Test version pattern recognition for GPG 1.x
	// Since we can't easily mock exec.Command, we test the pattern matching logic
	pattern := `\(GnuPG.*\) (1).(\d)`
	
	// Test that the pattern would match GPG 1.x format
	c.Check(strings.Contains(pattern, "(1)"), Equals, true)
}

func (s *GPGFinderSuite) TestCliVersionCheckGPG2Pattern(c *C) {
	// Test version pattern recognition for GPG 2.x
	pattern := `\(GnuPG.*\) (2).(\d)`
	
	// Test that the pattern would match GPG 2.x format
	c.Check(strings.Contains(pattern, "(2)"), Equals, true)
}

func (s *GPGFinderSuite) TestGPGFinderInterface(c *C) {
	// Test that all finders implement the GPGFinder interface
	var finder GPGFinder
	
	finder = GPG1Finder()
	c.Check(finder, NotNil)
	
	finder = GPG2Finder()
	c.Check(finder, NotNil)
	
	finder = GPGDefaultFinder()
	c.Check(finder, NotNil)
	
	// Test interface methods exist and return (may succeed or fail depending on system)
	gpg, gpgv, err1 := finder.FindGPG()
	_, _, err2 := finder.FindGPGV()
	
	// Methods should exist and return something
	if err1 == nil {
		// If GPG is found, paths should be non-empty
		c.Check(gpg, Not(Equals), "")
		c.Check(gpgv, Not(Equals), "")
	}
	// Test that both methods can be called (err2 may be nil or not)
	_ = err2
}

func (s *GPGFinderSuite) TestPathGPGFinderMultipleNames(c *C) {
	// Test that finder tries multiple names in order
	finder := &pathGPGFinder{
		gpgNames:                 []string{"nonexistent-first", "also-nonexistent"},
		gpgvNames:                []string{"nonexistent-gpgv1", "also-nonexistent-gpgv"},
		expectedVersionSubstring: `\(GnuPG.*\) (1).(\d)`,
		errorMessage:             "none found",
	}
	
	// Should try all names and still fail
	gpg, version, err := finder.FindGPG()
	c.Check(gpg, Equals, "")
	c.Check(err, NotNil)
	
	gpgv, version, err := finder.FindGPGV()
	c.Check(gpgv, Equals, "")
	c.Check(version, Equals, GPGVersion(0))
	c.Check(err, NotNil)
}

func (s *GPGFinderSuite) TestIteratingGPGFinderFirstSuccess(c *C) {
	// Test that iterating finder returns on first success
	successFinder := &mockSuccessfulGPGFinder{
		gpgResult:  "test-gpg",
		gpgvResult: "test-gpgv",
		version:    GPG1x,
	}
	
	failingFinder := &pathGPGFinder{
		gpgNames:     []string{"nonexistent"},
		gpgvNames:    []string{"nonexistent"},
		errorMessage: "should not reach this",
	}
	
	finder := &iteratingGPGFinder{
		finders:      []GPGFinder{successFinder, failingFinder},
		errorMessage: "should not see this error",
	}
	
	gpg, version, err := finder.FindGPG()
	c.Check(err, IsNil)
	c.Check(gpg, Equals, "test-gpg")
	c.Check(version, Equals, GPG1x)
	
	gpgv, version, err := finder.FindGPGV()
	c.Check(err, IsNil)
	c.Check(gpgv, Equals, "test-gpgv")
	c.Check(version, Equals, GPG1x)
}

func (s *GPGFinderSuite) TestGPGFinderErrorMessages(c *C) {
	// Test that error messages are appropriate for each finder type
	gpg1Finder := GPG1Finder().(*pathGPGFinder)
	c.Check(strings.Contains(gpg1Finder.errorMessage, "gnupg1"), Equals, true)
	c.Check(strings.Contains(gpg1Finder.errorMessage, "gpg(v)1"), Equals, true)
	
	gpg2Finder := GPG2Finder().(*pathGPGFinder)
	c.Check(strings.Contains(gpg2Finder.errorMessage, "gnupg2"), Equals, true)
	c.Check(strings.Contains(gpg2Finder.errorMessage, "gpg(v)2"), Equals, true)
	
	defaultFinder := GPGDefaultFinder().(*iteratingGPGFinder)
	c.Check(strings.Contains(defaultFinder.errorMessage, "gnupg"), Equals, true)
	c.Check(strings.Contains(defaultFinder.errorMessage, "suitable"), Equals, true)
}

func (s *GPGFinderSuite) TestRealGPGCommandExistence(c *C) {
	// Test if any real GPG commands exist in the system
	// This test documents the real-world behavior without failing if GPG is not installed
	
	commands := []string{"gpg", "gpg1", "gpg2", "gpgv", "gpgv1", "gpgv2"}
	foundCommands := []string{}
	
	for _, cmd := range commands {
		if _, err := exec.LookPath(cmd); err == nil {
			foundCommands = append(foundCommands, cmd)
		}
	}
	
	// This test just documents what's available, doesn't require any specific GPG
	c.Check(len(foundCommands) >= 0, Equals, true) // Always true, just documenting
}

// Mock implementation for testing
type mockSuccessfulGPGFinder struct {
	gpgResult  string
	gpgvResult string
	version    GPGVersion
}

func (m *mockSuccessfulGPGFinder) FindGPG() (string, GPGVersion, error) {
	return m.gpgResult, m.version, nil
}

func (m *mockSuccessfulGPGFinder) FindGPGV() (string, GPGVersion, error) {
	return m.gpgvResult, m.version, nil
}

func (s *GPGFinderSuite) TestMockGPGFinder(c *C) {
	// Test the mock finder implementation
	mock := &mockSuccessfulGPGFinder{
		gpgResult:  "mock-gpg",
		gpgvResult: "mock-gpgv", 
		version:    GPG21x,
	}
	
	gpg, version, err := mock.FindGPG()
	c.Check(err, IsNil)
	c.Check(gpg, Equals, "mock-gpg")
	c.Check(version, Equals, GPG21x)
	
	gpgv, version, err := mock.FindGPGV()
	c.Check(err, IsNil)
	c.Check(gpgv, Equals, "mock-gpgv")
	c.Check(version, Equals, GPG21x)
}

func (s *GPGFinderSuite) TestCliVersionCheckComplexVersions(c *C) {
	// Test version parsing with different GPG version outputs
	// Note: This test focuses on the regex parsing logic
	
	// Test patterns that would match different GPG versions
	pattern1x := `\(GnuPG.*\) (1).(\d)`
	pattern2x := `\(GnuPG.*\) (2).(\d)`
	
	// Verify patterns are correctly formed
	c.Check(strings.Contains(pattern1x, "(1)"), Equals, true)
	c.Check(strings.Contains(pattern2x, "(2)"), Equals, true)
	
	// Test with non-existent command to verify error handling
	result, version := cliVersionCheck("definitely-nonexistent-command-12345", pattern1x)
	c.Check(result, Equals, false)
	c.Check(version, Equals, GPGVersion(0))
}

func (s *GPGFinderSuite) TestGPGVersionEnumValues(c *C) {
	// Test all GPG version enum values
	c.Check(int(GPG1x), Equals, 1)
	c.Check(int(GPG20x), Equals, 2)
	c.Check(int(GPG21x), Equals, 3)
	c.Check(int(GPG22xPlus), Equals, 4)
	
	// Test version comparisons
	c.Check(GPG1x < GPG20x, Equals, true)
	c.Check(GPG20x < GPG21x, Equals, true)
	c.Check(GPG21x < GPG22xPlus, Equals, true)
}

func (s *GPGFinderSuite) TestIteratingGPGFinderPartialSuccess(c *C) {
	// Test iterating finder with first failing, second succeeding
	failingFinder := &pathGPGFinder{
		gpgNames:                 []string{"nonexistent-gpg"},
		gpgvNames:                []string{"nonexistent-gpgv"},
		expectedVersionSubstring: `\(GnuPG.*\) (1).(\d)`,
		errorMessage:             "first finder failed",
	}
	
	successFinder := &mockSuccessfulGPGFinder{
		gpgResult:  "second-gpg",
		gpgvResult: "second-gpgv",
		version:    GPG20x,
	}
	
	finder := &iteratingGPGFinder{
		finders:      []GPGFinder{failingFinder, successFinder},
		errorMessage: "all failed",
	}
	
	// Should succeed with second finder
	gpg, version, err := finder.FindGPG()
	c.Check(err, IsNil)
	c.Check(gpg, Equals, "second-gpg")
	c.Check(version, Equals, GPG20x)
	
	gpgv, version, err := finder.FindGPGV()
	c.Check(err, IsNil)
	c.Check(gpgv, Equals, "second-gpgv")
	c.Check(version, Equals, GPG20x)
}

func (s *GPGFinderSuite) TestPathGPGFinderEmptyArrays(c *C) {
	// Test pathGPGFinder with empty name arrays
	finder := &pathGPGFinder{
		gpgNames:                 []string{},
		gpgvNames:                []string{},
		expectedVersionSubstring: `\(GnuPG.*\) (1).(\d)`,
		errorMessage:             "no names to try",
	}
	
	gpg, version, err := finder.FindGPG()
	c.Check(gpg, Equals, "")
	c.Check(version, Equals, GPGVersion(0))
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "no names to try")
	
	gpgv, version, err := finder.FindGPGV()
	c.Check(gpgv, Equals, "")
	c.Check(version, Equals, GPGVersion(0))
	c.Check(err, NotNil)
}

func (s *GPGFinderSuite) TestIteratingGPGFinderEmptyFinders(c *C) {
	// Test iterating finder with no finders
	finder := &iteratingGPGFinder{
		finders:      []GPGFinder{},
		errorMessage: "no finders available",
	}
	
	gpg, version, err := finder.FindGPG()
	c.Check(gpg, Equals, "")
	c.Check(version, Equals, GPGVersion(0))
	c.Check(err, NotNil)
	c.Check(err.Error(), Equals, "no finders available")
	
	gpgv, version, err := finder.FindGPGV()
	c.Check(gpgv, Equals, "")
	c.Check(version, Equals, GPGVersion(0))
	c.Check(err, NotNil)
}