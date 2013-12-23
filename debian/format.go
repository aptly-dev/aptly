package debian

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// Stanza or paragraph of Debian control file
type Stanza map[string]string

// Copy returns copy of Stanza
func (s Stanza) Copy() (result Stanza) {
	result = make(Stanza, len(s))
	for k, v := range s {
		result[k] = v
	}
	return
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
			lastField = parts[0]
			stanza[lastField] = strings.TrimSpace(parts[1])
			_, lastFieldMultiline = multilineFields[lastField]
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
