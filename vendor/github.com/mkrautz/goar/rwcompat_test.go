// Copyright (c) 2011 Mikkel Krautz
// The use of this source code is goverened by a BSD-style
// license that can be found in the LICENSE-file.

package ar

// This file tests that our reader can read files written by our writer.

import (
	"bytes"
	"io"
	"testing"
)

type archiveTest struct {
	hdr  *Header
	data []byte
}

// Long filename test
var longFnArchive []archiveTest = []archiveTest{
	{
		&Header{
			Name:  "ReadWriteCompatFileName",
			Mode:  0755,
			Mtime: 0,
			Uid:   1000,
			Gid:   1000,
			Size:  6,
		},
		[]byte{0xf0, 0x00, 0xca, 0xfe, 0xba, 0xbe},
	},
}

// Short filename test (we don't special case short filenames,
// we always write out long-style filenames, but let's test it
// just to make sure)
var shortFnArchive []archiveTest = []archiveTest{
	{
		&Header{
			Name:  "a",
			Mode:  0755,
			Mtime: 0,
			Uid:   1000,
			Gid:   1000,
			Size:  2,
		},
		[]byte{'a', '\n'},
	},
}

func headerCmp(hdr1 *Header, hdr2 *Header) bool {
	if hdr1 == nil || hdr2 == nil {
		return false
	}
	if hdr1.Name != hdr2.Name {
		return false
	}
	if hdr1.Mode != hdr2.Mode {
		return false
	}
	if hdr1.Mtime != hdr2.Mtime {
		return false
	}
	if hdr1.Uid != hdr2.Uid {
		return false
	}
	if hdr1.Gid != hdr2.Gid {
		return false
	}
	if hdr1.Size != hdr2.Size {
		return false
	}
	return true
}

func testRwCompat(t *testing.T, testArchive []archiveTest) {
	buf := new(bytes.Buffer)

	w := NewWriter(buf)
	for _, testEntry := range testArchive {
		err := w.WriteHeader(testEntry.hdr)
		if err != nil {
			t.Fatal(err)
		}
		_, err = w.Write(testEntry.data)
		if err != nil {
			t.Fatal(err)
		}
	}
	err := w.Close()
	if err != nil {
		t.Fatal(err)
	}

	rbuf := bytes.NewBuffer(buf.Bytes())
	r := NewReader(rbuf)
	for _, testEntry := range testArchive {
		hdr, err := r.Next()
		if err != nil {
			t.Fatal(err)
		}
		if !headerCmp(hdr, testEntry.hdr) {
			t.Fatalf("header mismatch:\nread = %v\norig = %v", hdr, testEntry.hdr)
		}
		fbuf := make([]byte, hdr.Size)
		_, err = io.ReadFull(r, fbuf)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(fbuf, testEntry.data) {
			t.Fatalf("data mismatch\nread = %v\norig = %v", fbuf, testEntry.data)
		}
	}
}

func TestReadWriteCompatLongFn(t *testing.T) {
	testRwCompat(t, longFnArchive)
}

func TestReadWriteCompatShortFn(t *testing.T) {
	testRwCompat(t, shortFnArchive)
}
