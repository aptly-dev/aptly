// Copyright (c) 2011 Mikkel Krautz
// The use of this source code is goverened by a BSD-style
// license that can be found in the LICENSE-file.

package ar

import (
	"bytes"
	"testing"
)

func TestLongWrite(t *testing.T) {
	buf := new(bytes.Buffer)
	w := NewWriter(buf)
	err := w.WriteHeader(&Header{
			Name:  "/",
			Mode:  0,
			Mtime: 1315607407,
			Uid:   0,
			Gid:   0,
			Size:  4,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write([]byte{1, 2, 3, 4, 5})
	if err != ErrWriteTooLong {
		t.Fatalf("expected ErrWriteTooLong, got %v", err)
	}
}

func TestWriteAfterClose(t *testing.T) {
	buf := new(bytes.Buffer)
	w := NewWriter(buf)
	err := w.WriteHeader(&Header{
			Name:  "/",
			Mode:  0,
			Mtime: 1315607407,
			Uid:   0,
			Gid:   0,
			Size:  4,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write([]byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	err = w.Close()
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write([]byte{4})
	if err != ErrWriteAfterClose {
		t.Fatalf("expected ErrWriteAfterClose, got %v", err)
	}
}
