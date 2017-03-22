// Copyright (c) 2011 Mikkel Krautz
// The use of this source code is goverened by a BSD-style
// license that can be found in the LICENSE-file.

// Package ar implements reading and writing of ar archives.
// It supports reading archives in the GNU and BSD formats, but
// only supports writing in the BSD format.
package ar

import "errors"

// A Header represents a single file header in an ar archive.
type Header struct {
	Name  string
	Mode  int64
	Uid   int
	Gid   int
	Size  int64
	Mtime int64
}

var (
	globalHeader          = "!<arch>\n"
	bsdLongFileNamePrefix = "#1/"
	fileHeaderMagic       = "`\n"
)

var (
	ErrMissingGlobalHeader = errors.New("ar: missing global header")
	ErrFileHeader          = errors.New("ar: invalid ar file header")
	ErrWriteAfterClose     = errors.New("ar: write after close")
	ErrWriteTooLong        = errors.New("ar: write too long")
)

type skippingWriter struct{}

func (sw skippingWriter) Write(buf []byte) (int, error) {
	return len(buf), nil
}

// nulTerminated returns a nul terminated string read from buf.
func nulTerminated(buf []byte) string {
	for i := 0; i < len(buf); i++ {
		if buf[i] == 0 {
			return string(buf[0:i])
		}
	}
	return string(buf)
}

// gnuArString reads a linefeed terminated string from buf.
// If a linefeed character is not found in buf, the function
// returns an error.
func gnuArString(buf []byte) (string, error) {
	for i := 0; i < len(buf); i++ {
		if buf[i] == '\n' {
			return string(buf[:i]), nil
		}
	}
	return "", errors.New("ar: missing linefeed in parsing ar string")
}

// arString reads a whitespace terminated string from the string in.
func arString(in string) string {
	for i, rune := range in {
		if rune == ' ' || rune == '\t' {
			return in[:i]
		}
	}
	return in
}

// Encode a string to the whitespace-padded format used in the ar header.
func encodeArString(in string, total int) string {
	remain := total - len(in)
	var spaces []byte
	for i := 0; i < remain; i++ {
		spaces = append(spaces, ' ')
	}
	return in + string(spaces)
}
