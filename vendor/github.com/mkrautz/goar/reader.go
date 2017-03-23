// Copyright (c) 2011 Mikkel Krautz
// The use of this source code is goverened by a BSD-style
// license that can be found in the LICENSE-file.

package ar

import (
	"errors"
	"io"
	"strconv"
	"strings"
)

// A Reader provides sequential access to the contents of a BSD or GNU-style ar archive.
// An archive file consists of a sequence of files.
// The Next method advances to the next file in the archive (including the first).
// After Next has returned a header, the Reader can be treated as an io.Reader to
// access the data of the file described by the header received from Next.
//
// Example:
//	tr := ar.NewReader(r)
//	for {
//		hdr, err := tr.Next()
//		if err == io.EOF {
//			// end of archive
//			break
//		}
//		if err != nil {
//			// handle error
//		}
//		io.Copy(data, tr)
//	}
type Reader struct {
	r          io.Reader
	offset     int64
	dataRemain int64
	gnuLongFn  []byte
}

// NewReader creates a new Reader reading from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{r, 0, 0, nil}
}

// Next advances to the next entry in the archive.
func (ar *Reader) Next() (hdr *Header, err error) {
	// If this is our first read, we should check whether a global
	// ar header is present.
	if ar.offset == 0 {
		ghdr := make([]byte, len(globalHeader))
		nread, err := io.ReadFull(ar.r, ghdr)
		if err != nil {
			return nil, err
		}
		ar.offset += int64(nread)
		if globalHeader != string(ghdr) {
			return nil, ErrMissingGlobalHeader
		}
	}

	// If an entry wasn't fully read, skip the remaining bytes
	if ar.dataRemain > 0 {
		sw := skippingWriter{}
		ncopied, err := io.CopyN(sw, ar.r, ar.dataRemain)
		if err == io.EOF || err == nil {
			ar.offset += ncopied
			ar.dataRemain -= ncopied
			if ar.dataRemain > 0 {
				return nil, errors.New("ar: skip failed")
			}
		} else if err != nil {
			return nil, err
		}
	}

	// Read a file header from the archive.
	hdr, err = ar.consumeHeader()
	if err != nil {
		return nil, err
	}

	// If the consumed header is a GNU long file name section,
	// read its filename table and update the Reader struct with it.
	if hdr.Name == "//" {
		// Return an error if we've already read a GNU long filename
		// section.
		if ar.gnuLongFn != nil {
			return nil, errors.New("ar: malformed archive, duplicate gnu long filename sections")
		}

		ar.dataRemain = hdr.Size
		buf := make([]byte, int(hdr.Size))
		_, err = io.ReadFull(ar, buf)
		// We expect the GNU long filename section
		// to be as long as noted in the header.
		if err != nil {
			return nil, err
		}
		ar.gnuLongFn = buf

		// The special header has been consumed.
		// Read the next file header in the file so we can return
		// that to the user.
		hdr, err = ar.consumeHeader()
		if err != nil {
			return nil, err
		}
	}

	ar.dataRemain = hdr.Size
	return hdr, nil
}

// Read reads from the current entry in the archive.
// It returns 0, io.EOF when it reaches the end of that entry,
// until Next is called to advance to the next entry.
func (ar *Reader) Read(b []byte) (n int, err error) {
	if ar.dataRemain == 0 {
		return 0, io.EOF
	}
	if int64(len(b)) > ar.dataRemain {
		b = b[:ar.dataRemain]
	}
	n, err = ar.r.Read(b)
	ar.offset += int64(n)
	ar.dataRemain -= int64(n)
	if ar.dataRemain == 0 {
		err = io.EOF
	}
	return
}

func (ar *Reader) consumeHeader() (*Header, error) {
	// Data sections are required to always end on a 2-byte boundary.
	// Simply check if we're at a 2-byte offset before consuming a new
	// file header. If not, consume the padding byte and check that it
	// is a '/n' like we expect.
	if ar.offset%2 != 0 {
		lineFeed := make([]byte, 1)
		_, err := ar.r.Read(lineFeed)
		if err != nil {
			if lineFeed[0] != '\n' {
				return nil, errors.New("ar: alignment byte read, not '\n'")
			}
		}
		ar.offset += 1
	}

	fhdr := make([]byte, 60)
	nread, err := io.ReadFull(ar.r, fhdr)
	if err != nil {
		return nil, err
	}
	ar.offset += int64(nread)

	hdr := &Header{}
	fileName := arString(string(fhdr[0:16]))
	mtime := arString(string(fhdr[16:28]))
	uid := arString(string(fhdr[28:34]))
	gid := arString(string(fhdr[34:40]))
	mode := arString(string(fhdr[40:48]))
	size := arString(string(fhdr[48:58]))
	magic := arString(string(fhdr[58:60]))

	if magic != fileHeaderMagic {
		return nil, ErrFileHeader
	}

	if mtime != "" {
		hdr.Mtime, err = strconv.ParseInt(mtime, 10, 64)
		if err != nil {
			return nil, err
		}
	}

	if uid != "" {
		hdr.Uid, err = strconv.Atoi(uid)
		if err != nil {
			return nil, err
		}
	}

	if gid != "" {
		hdr.Gid, err = strconv.Atoi(gid)
		if err != nil {
			return nil, err
		}
	}

	hdr.Size, err = strconv.ParseInt(size, 10, 64)
	if err != nil {
		return nil, err
	}

	if mode != "" {
		hdr.Mode, err = strconv.ParseInt(mode, 8, 64)
		if err != nil {
			return nil, err
		}
	}

	// GNU-style ar archives use '/' as a filename terminator for everything
	// but special sections (sections that start with a '/'), so we strip trailing
	// slashes from all filenames that do not start with a slash themselves.
	if len(fileName) > 0 && fileName[0] != '/' && fileName[len(fileName)-1] == '/' {
		fileName = fileName[:len(fileName)-1]
	}

	// The file name is stored as a BSD long filename
	// That is, the filename is stored directly after the file header, as
	// part of the data section.
	if strings.HasPrefix(fileName, bsdLongFileNamePrefix) {
		fnLengthStr := arString(fileName[len(bsdLongFileNamePrefix):])
		fnLength, err := strconv.Atoi(fnLengthStr)
		if err != nil {
			return nil, err
		}
		if int64(fnLength) > hdr.Size {
			return nil, errors.New("ar: invalid bsd long filename in file")
		}
		longFn := make([]byte, fnLength)
		nread, err = io.ReadFull(ar.r, longFn)
		if err != nil {
			return nil, err
		}
		ar.offset += int64(nread)
		hdr.Size -= int64(nread)
		hdr.Name = nulTerminated(longFn)

		// The file name is stored as a GNU long filename
	} else if fhdr[0] == '/' && fhdr[1] >= '0' && fhdr[1] <= '9' {
		// We must have read a GNU-style long filename section for this lookup
		// to succeed.
		if ar.gnuLongFn == nil {
			return nil, errors.New("ar: encountered gnu-style long fn without corresponding long fn section")
		}
		gnuOffset, err := strconv.ParseInt(fileName[1:], 10, 64)
		if err != nil {
			return nil, err
		}
		if gnuOffset <= int64(len(ar.gnuLongFn)) {
			fnStr, err := gnuArString(ar.gnuLongFn[gnuOffset:])
			if err != nil {
				return nil, err
			}
			if fnStr[len(fnStr)-1] != '/' {
				return nil, errors.New("ar: gnu long filename is not terminated")
			}
			hdr.Name = fnStr[:len(fnStr)-1]
		} else {
			// The offset overflows our long filename section
			return nil, errors.New("ar: gnu long filename lookup out of bounds")
		}

		// Regular short file name
	} else {
		hdr.Name = fileName
	}

	return hdr, nil
}
