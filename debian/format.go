package debian

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// Stanza or paragraph of Debian control file
type Stanza map[string]string

// Canonical order of fields in stanza
var canocialOrder = []string{"Origin", "Label", "Suite", "Package", "Version", "Installed-Size", "Priority", "Section", "Maintainer",
	"Architecture", "Codename", "Date", "Architectures", "Components", "Description", "MD5sum", "MD5Sum", "SHA1", "SHA256"}

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
func (s Stanza) WriteTo(w *bufio.Writer) error {
	for _, field := range canocialOrder {
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
