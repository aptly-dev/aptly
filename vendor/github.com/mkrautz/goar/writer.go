// Copyright (c) 2011 Mikkel Krautz
// The use of this source code is goverened by a BSD-style
// license that can be found in the LICENSE-file.

package ar

import (
	"io"
	"strconv"
)

// A Writer provides sequential writing of an ar archive in BSD format. It does not support
// writing in the GNU format, since the GNU-style extended filenames cannot be written sequentially.
// The BSD ar format is widely compatible with most modern ar readers out there.
//
// An ar archive consists of a sequence of files. Call WriteHeader to begin a new file,
// and then call Write to supply that file's data, writing at most hdr.Size bytes in total.
//
// Example:
//	aw := ar.NewWriter(w)
//	hdr := new(ar.Header)
//	hdr.Size = length of data in bytes
//	// populate other hdr fields as desired
//	if err := aw.WriteHeader(hdr); err != nil {
//		// handle error
//	}
//	io.Copy(tw, data)
//	tw.Close()
type Writer struct {
	w          io.Writer
	offset     int64
	dataRemain int64
	closed     bool
}

// NewWriter creates a new Writer writing to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w, 0, 0, false}
}

// Closes the ar achive, flushing any unwritten data to the underlying writer.
func (aw *Writer) Close() error {
	err := aw.Flush()
	if err != nil {
		return err
	}
	aw.closed = true
	return nil
}

// Flush finishes writing the current file (optional).
func (aw *Writer) Flush() error {
	if aw.closed {
		return ErrWriteAfterClose
	}
	if aw.offset%2 != 0 {
		_, err := io.WriteString(aw.w, "\n")
		aw.offset += int64(1)
		if err != nil {
			return err
		}
	}
	return nil
}

// Write writes the current entry in the ar archive. Write returns the error ErrWriteTooLong
// if more than hdr.Size bytes are written following a call to WriteHeader.
func (aw *Writer) Write(b []byte) (n int, err error) {
	if aw.closed {
		return 0, ErrWriteAfterClose
	}
	// Overflow check
	tooLong := false
	if int64(len(b)) > aw.dataRemain {
		b = b[0:aw.dataRemain]
		tooLong = true
	}
	n, err = aw.w.Write(b)
	aw.dataRemain -= int64(n)
	// Warn if the write would have overflowed the
	// space set aside for the provided data.
	if err == nil && tooLong {
		err = ErrWriteTooLong
		return
	}
	return
}

// WriteHeader writes hdr and prepares to accept the file's content.  WriteHeader calls Flush to
// correctly pad the last written file. Calling after WriteHeader a Close will return ErrWriteAfterClose.
func (aw *Writer) WriteHeader(hdr *Header) (err error) {
	if aw.closed {
		return ErrWriteAfterClose
	}

	// Flush previous data write
	err = aw.Flush()
	if err != nil {
		return err
	}

	// If we're at the beginning of the writer, write
	// the global header.
	if aw.offset == 0 {
		nwritten, err := io.WriteString(aw.w, globalHeader)
		aw.offset += int64(nwritten)
		if err != nil {
			return err
		}
	}

	var (
		newName string
		newSize int64
	)

	longFn := len(hdr.Name) > 15
	if longFn {
		newName = bsdLongFileNamePrefix + strconv.Itoa(len(hdr.Name))
		newSize = hdr.Size + int64(len(hdr.Name))
	} else {
		newName = hdr.Name
		newSize = hdr.Size
	}

	nwritten, err := io.WriteString(aw.w, encodeArString(newName, 16))
	aw.offset += int64(nwritten)
	if err != nil {
		return err
	}

	nwritten, err = io.WriteString(aw.w, encodeArString(strconv.FormatInt(hdr.Mtime, 10), 12))
	aw.offset += int64(nwritten)
	if err != nil {
		return err
	}

	nwritten, err = io.WriteString(aw.w, encodeArString(strconv.Itoa(hdr.Uid), 6))
	aw.offset += int64(nwritten)
	if err != nil {
		return err
	}

	nwritten, err = io.WriteString(aw.w, encodeArString(strconv.Itoa(hdr.Gid), 6))
	aw.offset += int64(nwritten)
	if err != nil {
		return err
	}

	nwritten, err = io.WriteString(aw.w, encodeArString(strconv.FormatInt(hdr.Mode, 8), 8))
	aw.offset += int64(nwritten)
	if err != nil {
		return err
	}

	nwritten, err = io.WriteString(aw.w, encodeArString(strconv.FormatInt(newSize, 10), 10))
	aw.offset += int64(nwritten)
	if err != nil {
		return err
	}

	nwritten, err = io.WriteString(aw.w, fileHeaderMagic)
	aw.offset += int64(nwritten)
	if err != nil {
		return err
	}

	if longFn {
		nwritten, err = io.WriteString(aw.w, hdr.Name)
		aw.offset += int64(nwritten)
		aw.dataRemain = newSize - int64(nwritten)
		if err != nil {
			return err
		}
	} else {
		aw.dataRemain = newSize
	}

	return nil
}
