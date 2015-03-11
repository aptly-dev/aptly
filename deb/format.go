package deb

import (
	"bufio"
	"errors"
	"io"
	"strings"
	"unicode"
)

// Stanza or paragraph of Debian control file
type Stanza map[string]string

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
		"Architectures",
		"Architecture",
		"Components",
		"Component",
		"Description",
		"MD5Sum",
		"SHA1",
		"SHA256",
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
)

// Copy returns copy of Stanza
func (s Stanza) Copy() (result Stanza) {
	result = make(Stanza, len(s))
	for k, v := range s {
		result[k] = v
	}
	return
}

// Write single field from Stanza to writer
func writeField(w *bufio.Writer, field, value string) (err error) {
	_, multiline := multilineFields[field]

	if !multiline {
		_, err = w.WriteString(field + ": " + value + "\n")
	} else {
		if !strings.HasSuffix(value, "\n") {
			value = value + "\n"
		}
		_, err = w.WriteString(field + ":" + value)
	}

	return
}

// WriteTo saves stanza back to stream, modifying itself on the fly
func (s Stanza) WriteTo(w *bufio.Writer, isSource, isRelease bool) error {
	canonicalOrder := canonicalOrderBinary
	if isSource {
		canonicalOrder = canonicalOrderSource
	}
	if isRelease {
		canonicalOrder = canonicalOrderRelease
	}

	for _, field := range canonicalOrder {
		value, ok := s[field]
		if ok {
			delete(s, field)
			err := writeField(w, field, value)
			if err != nil {
				return err
			}
		}
	}

	for field, value := range s {
		err := writeField(w, field, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// Parsing errors
var (
	ErrMalformedStanza = errors.New("malformed stanza syntax")
)

var multilineFields = make(map[string]bool)

func init() {
	multilineFields["Description"] = true
	multilineFields["Files"] = true
	multilineFields["Changes"] = true
	multilineFields["Checksums-Sha1"] = true
	multilineFields["Checksums-Sha256"] = true
	multilineFields["Package-List"] = true
	multilineFields["SHA256"] = true
	multilineFields["SHA1"] = true
	multilineFields["MD5Sum"] = true
}

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
	scanner *bufio.Scanner
}

// NewControlFileReader creates ControlFileReader, it wraps with buffering
func NewControlFileReader(r io.Reader) *ControlFileReader {
	return &ControlFileReader{scanner: bufio.NewScanner(bufio.NewReaderSize(r, 32768))}
}

// ReadStanza reeads one stanza from control file
func (c *ControlFileReader) ReadStanza() (Stanza, error) {
	stanza := make(Stanza, 32)
	lastField := ""
	lastFieldMultiline := false

	for c.scanner.Scan() {
		line := c.scanner.Text()

		// Current stanza ends with empty line
		if line == "" {
			if len(stanza) > 0 {
				return stanza, nil
			}
			continue
		}

		if line[0] == ' ' || line[0] == '\t' {
			if lastFieldMultiline {
				stanza[lastField] += line + "\n"
			} else {
				stanza[lastField] += strings.TrimSpace(line)
			}
		} else {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				return nil, ErrMalformedStanza
			}
			lastField = canonicalCase(parts[0])
			_, lastFieldMultiline = multilineFields[lastField]
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
