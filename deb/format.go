package deb

import (
	"bufio"
	"errors"
	"io"
	"sort"
	"strings"
	"unicode"
)

// Stanza or paragraph of Debian control file
type Stanza map[string]string

// MaxFieldSize is maximum stanza field size in bytes
const MaxFieldSize = 2 * 1024 * 1024

// Canonical order of fields in stanza
// Taken from: http://bazaar.launchpad.net/~ubuntu-branches/ubuntu/vivid/apt/vivid/view/head:/apt-pkg/tagfile.cc#L504
var (
	canonicalOrderRelease = []string{
		"Origin",
		"Label",
		"Archive",
		"Suite",
		"Version",
		"Codename",
		"Date",
		"NotAutomatic",
		"ButAutomaticUpgrades",
		"Architectures",
		"Architecture",
		"Components",
		"Component",
		"Description",
		"MD5Sum",
		"SHA1",
		"SHA256",
		"SHA512",
	}

	canonicalOrderBinary = []string{
		"Package",
		"Essential",
		"Status",
		"Priority",
		"Section",
		"Installed-Size",
		"Maintainer",
		"Original-Maintainer",
		"Architecture",
		"Source",
		"Version",
		"Replaces",
		"Provides",
		"Depends",
		"Pre-Depends",
		"Recommends",
		"Suggests",
		"Conflicts",
		"Breaks",
		"Conffiles",
		"Filename",
		"Size",
		"MD5Sum",
		"MD5sum",
		"SHA1",
		"SHA256",
		"SHA512",
		"Description",
	}

	canonicalOrderSource = []string{
		"Package",
		"Source",
		"Binary",
		"Version",
		"Priority",
		"Section",
		"Maintainer",
		"Original-Maintainer",
		"Build-Depends",
		"Build-Depends-Indep",
		"Build-Conflicts",
		"Build-Conflicts-Indep",
		"Architecture",
		"Standards-Version",
		"Format",
		"Directory",
		"Files",
	}
	canonicalOrderInstaller = []string{
		"",
	}
)

// Copy returns copy of Stanza
func (s Stanza) Copy() (result Stanza) {
	result = make(Stanza, len(s))
	for k, v := range s {
		result[k] = v
	}
	return
}

func isMultilineField(field string, isRelease bool) bool {
	switch field {
	// file without a section
	case "":
		return true
	case "Description":
		return true
	case "Files":
		return true
	case "Changes":
		return true
	case "Checksums-Sha1":
		return true
	case "Checksums-Sha256":
		return true
	case "Checksums-Sha512":
		return true
	case "Package-List":
		return true
	case "MD5Sum":
		return isRelease
	case "SHA1":
		return isRelease
	case "SHA256":
		return isRelease
	case "SHA512":
		return isRelease
	}
	return false
}

// Write single field from Stanza to writer.
//
// nolint: interfacer
func writeField(w *bufio.Writer, field, value string, isRelease bool) (err error) {
	if !isMultilineField(field, isRelease) {
		_, err = w.WriteString(field + ": " + value + "\n")
	} else {
		if field != "" && !strings.HasSuffix(value, "\n") {
			value = value + "\n"
		}

		if field != "Description" && field != "" {
			value = "\n" + value
		}

		if field != "" {
			_, err = w.WriteString(field + ":" + value)
		} else {
			_, err = w.WriteString(value)
		}
	}

	return
}

// WriteTo saves stanza back to stream, modifying itself on the fly
func (s Stanza) WriteTo(w *bufio.Writer, isSource, isRelease, isInstaller bool) error {
	canonicalOrder := canonicalOrderBinary
	if isSource {
		canonicalOrder = canonicalOrderSource
	}
	if isRelease {
		canonicalOrder = canonicalOrderRelease
	}
	if isInstaller {
		canonicalOrder = canonicalOrderInstaller
	}

	for _, field := range canonicalOrder {
		value, ok := s[field]
		if ok {
			delete(s, field)
			err := writeField(w, field, value, isRelease)
			if err != nil {
				return err
			}
		}
	}

	// no extra fields in installer
	if !isInstaller {
		// Print extra fields in deterministic order (alphabetical)
		keys := make([]string, len(s))
		i := 0
		for field := range s {
			keys[i] = field
			i++
		}
		sort.Strings(keys)
		for _, field := range keys {
			err := writeField(w, field, s[field], isRelease)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Parsing errors
var (
	ErrMalformedStanza = errors.New("malformed stanza syntax")
)

func canonicalCase(field string) string {
	upper := strings.ToUpper(field)
	switch upper {
	case "SHA1", "SHA256", "SHA512":
		return upper
	case "MD5SUM":
		return "MD5Sum"
	case "NOTAUTOMATIC":
		return "NotAutomatic"
	case "BUTAUTOMATICUPGRADES":
		return "ButAutomaticUpgrades"
	}

	startOfWord := true

	return strings.Map(func(r rune) rune {
		if startOfWord {
			startOfWord = false
			return unicode.ToUpper(r)
		}

		if r == '-' {
			startOfWord = true
		}

		return unicode.ToLower(r)
	}, field)
}

// ControlFileReader implements reading of control files stanza by stanza
type ControlFileReader struct {
	scanner     *bufio.Scanner
	isRelease   bool
	isInstaller bool
}

// NewControlFileReader creates ControlFileReader, it wraps with buffering
func NewControlFileReader(r io.Reader, isRelease, isInstaller bool) *ControlFileReader {
	scnr := bufio.NewScanner(bufio.NewReaderSize(r, 32768))
	scnr.Buffer(nil, MaxFieldSize)

	return &ControlFileReader{
		scanner:     scnr,
		isRelease:   isRelease,
		isInstaller: isInstaller,
	}
}

// ReadStanza reeads one stanza from control file
func (c *ControlFileReader) ReadStanza() (Stanza, error) {
	stanza := make(Stanza, 32)
	lastField := ""
	lastFieldMultiline := c.isInstaller

	for c.scanner.Scan() {
		line := c.scanner.Text()

		// Current stanza ends with empty line
		if line == "" {
			if len(stanza) > 0 {
				return stanza, nil
			}
			continue
		}

		if line[0] == ' ' || line[0] == '\t' || c.isInstaller {
			if lastFieldMultiline {
				stanza[lastField] += line + "\n"
			} else {
				stanza[lastField] += " " + strings.TrimSpace(line)
			}
		} else {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				return nil, ErrMalformedStanza
			}
			lastField = canonicalCase(parts[0])
			lastFieldMultiline = isMultilineField(lastField, c.isRelease)
			if lastFieldMultiline {
				stanza[lastField] = parts[1]
				if parts[1] != "" {
					stanza[lastField] += "\n"
				}
			} else {
				stanza[lastField] = strings.TrimSpace(parts[1])
			}
		}
	}
	if err := c.scanner.Err(); err != nil {
		return nil, err
	}
	if len(stanza) > 0 {
		return stanza, nil
	}
	return nil, nil
}
