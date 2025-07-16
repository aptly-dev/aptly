package utils

import (
	"strings"
	
	. "gopkg.in/check.v1"
)

type SanitizeSuite struct{}

var _ = Suite(&SanitizeSuite{})

func (s *SanitizeSuite) TestSanitizePath(c *C) {
	// Test various path scenarios based on actual implementation
	// The function removes "..", "$", "`" and leading "/"
	testCases := []struct {
		input    string
		expected string
		desc     string
	}{
		{
			input:    "normal/path/file.txt",
			expected: "normal/path/file.txt",
			desc:     "normal path unchanged",
		},
		{
			input:    "../../../etc/passwd",
			expected: "etc/passwd",
			desc:     ".. removed from path",
		},
		{
			input:    "/absolute/path",
			expected: "absolute/path",
			desc:     "leading slash removed",
		},
		{
			input:    "path$with$dollar",
			expected: "pathwithdollar",
			desc:     "dollar signs removed",
		},
		{
			input:    "path`with`backticks",
			expected: "pathwithbackticks",
			desc:     "backticks removed",
		},
		{
			input:    "$HOME/.ssh/id_rsa",
			expected: "HOME/.ssh/id_rsa",
			desc:     "environment variable syntax removed",
		},
		{
			input:    "`echo malicious`",
			expected: "echo malicious",
			desc:     "command substitution removed",
		},
		{
			input:    "path/../other",
			expected: "path//other",
			desc:     "internal .. removed leaving double slash",
		},
		{
			input:    "",
			expected: "",
			desc:     "empty path stays empty",
		},
		{
			input:    "///multiple/leading/slashes",
			expected: "multiple/leading/slashes",
			desc:     "multiple leading slashes removed",
		},
		{
			input:    "test$../$../../../etc/passwd",
			expected: "test////etc/passwd",
			desc:     "combined $ and .. removal",
		},
		{
			input:    "/",
			expected: "",
			desc:     "single slash becomes empty",
		},
		{
			input:    "//",
			expected: "",
			desc:     "double slash becomes empty",
		},
		{
			input:    "valid/path/without/issues",
			expected: "valid/path/without/issues",
			desc:     "valid path unchanged",
		},
	}

	for _, tc := range testCases {
		result := SanitizePath(tc.input)
		c.Check(result, Equals, tc.expected, Commentf("Test case: %s", tc.desc))
	}
}

func (s *SanitizeSuite) TestSanitizePathSecurity(c *C) {
	// Test specific security scenarios
	securityTests := []struct {
		input string
		desc  string
	}{
		{"../../../../../../../../etc/shadow", "deep traversal"},
		{"../../../.ssh/id_rsa", "hidden file access"},
		{"./../../../root/.bashrc", "dotfile access"},
		{"$PATH/../../../../etc/hosts", "env var with traversal"},
		{"`id`/../../../etc/passwd", "command injection with traversal"},
		{"${HOME}/.aws/credentials", "env var syntax"},
		{"$(whoami)/../sensitive", "command substitution"},
	}

	for _, tc := range securityTests {
		result := SanitizePath(tc.input)
		// After sanitization, should not contain dangerous patterns
		c.Check(strings.Contains(result, ".."), Equals, false, Commentf("Security test failed for: %s, got: %s", tc.desc, result))
		c.Check(strings.Contains(result, "$"), Equals, false, Commentf("Dollar sign present for: %s, got: %s", tc.desc, result))
		c.Check(strings.Contains(result, "`"), Equals, false, Commentf("Backtick present for: %s, got: %s", tc.desc, result))
		// Should not start with /
		if len(result) > 0 {
			c.Check(result[0] != '/', Equals, true, Commentf("Absolute path for: %s, got: %s", tc.desc, result))
		}
	}
}