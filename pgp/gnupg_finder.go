package pgp

import (
	"errors"
	"os/exec"
	"regexp"
	"strings"
)

// GPGVersion stores discovered GPG version
type GPGVersion int

// GPG version as discovered
const (
	GPG1x      GPGVersion = 1
	GPG20x     GPGVersion = 2
	GPG21xPlus GPGVersion = 3
)

var gpgVersionRegex = regexp.MustCompile(`\(GnuPG\) (\d)\.(\d)`)

// GPGFinder implement search for gpg executables and returns version of discovered executables
type GPGFinder interface {
	FindGPG() (gpg string, version GPGVersion, err error)
	FindGPGV() (gpgv string, version GPGVersion, err error)
}

type pathGPGFinder struct {
	gpgNames     []string
	gpgvNames    []string
	errorMessage string

	expectedVersionSubstring string
}

type iteratingGPGFinder struct {
	finders      []GPGFinder
	errorMessage string
}

// GPGDefaultFinder looks for GPG1 first, but falls back to GPG2 if GPG1 is not available
func GPGDefaultFinder() GPGFinder {
	return &iteratingGPGFinder{
		finders:      []GPGFinder{GPG1Finder(), GPG2Finder()},
		errorMessage: "Couldn't find a suitable gpg executable. Make sure gnupg is installed",
	}
}

// GPG1Finder looks for GnuPG1.x only
func GPG1Finder() GPGFinder {
	return &pathGPGFinder{
		gpgNames:                 []string{"gpg", "gpg1"},
		gpgvNames:                []string{"gpgv", "gpgv1"},
		expectedVersionSubstring: "(GnuPG) 1.",
		errorMessage:             "Couldn't find a suitable gpg executable. Make sure gnupg1 is available as either gpg(v) or gpg(v)1 in $PATH",
	}
}

// GPG2Finder looks for GnuPG2.x only
func GPG2Finder() GPGFinder {
	return &pathGPGFinder{
		gpgNames:                 []string{"gpg", "gpg2"},
		gpgvNames:                []string{"gpgv", "gpgv2"},
		expectedVersionSubstring: "(GnuPG) 2.",
		errorMessage:             "Couldn't find a suitable gpg executable. Make sure gnupg2 is available as either gpg(v) or gpg(v)2 in $PATH",
	}
}

func (pgf *pathGPGFinder) FindGPG() (gpg string, version GPGVersion, err error) {
	for _, cmd := range pgf.gpgNames {
		var result bool
		result, version = cliVersionCheck(cmd, pgf.expectedVersionSubstring)
		if result {
			gpg = cmd
			break
		}
	}

	if gpg == "" {
		err = errors.New(pgf.errorMessage)
	}

	return
}

func (pgf *pathGPGFinder) FindGPGV() (gpgv string, version GPGVersion, err error) {
	for _, cmd := range pgf.gpgvNames {
		var result bool
		result, version = cliVersionCheck(cmd, pgf.expectedVersionSubstring)
		if result {
			gpgv = cmd
			break
		}
	}

	if gpgv == "" {
		err = errors.New(pgf.errorMessage)
	}

	return
}

func (it *iteratingGPGFinder) FindGPG() (gpg string, version GPGVersion, err error) {
	for _, finder := range it.finders {
		gpg, version, err = finder.FindGPG()
		if err == nil {
			return
		}
	}

	err = errors.New(it.errorMessage)

	return
}

func (it *iteratingGPGFinder) FindGPGV() (gpg string, version GPGVersion, err error) {
	for _, finder := range it.finders {
		gpg, version, err = finder.FindGPGV()
		if err == nil {
			return
		}
	}

	err = errors.New(it.errorMessage)

	return
}

func cliVersionCheck(cmd string, marker string) (result bool, version GPGVersion) {
	output, err := exec.Command(cmd, "--version").CombinedOutput()
	if err != nil {
		return
	}

	strOutput := string(output)
	result = strings.Contains(strOutput, marker)

	version = GPG21xPlus
	matches := gpgVersionRegex.FindStringSubmatch(strOutput)
	if matches != nil {
		if matches[1] == "1" {
			version = GPG1x
		} else if matches[1] == "2" && matches[2] == "0" {
			version = GPG20x
		}
	}

	return
}
