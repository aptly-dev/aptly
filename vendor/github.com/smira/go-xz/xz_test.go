package xz

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestDecompress(T *testing.T) {
	orig, err := os.Open("testdata/spec")
	if err != nil {
		T.Fatal(err)
	}
	defer orig.Close()

	expected := &bytes.Buffer{}
	_, err = io.Copy(expected, orig)
	if err != nil {
		T.Fatal(err)
	}

	source, err := os.Open("testdata/spec.xz")
	if err != nil {
		T.Fatal(err)
	}
	defer source.Close()

	r, err := NewReader(source)
	if err != nil {
		T.Fatal(err)
	}

	buf := &bytes.Buffer{}
	n, err := io.Copy(buf, r)
	if n != int64(expected.Len()) {
		T.Fail()
	}

	if bytes.Compare(buf.Bytes(), expected.Bytes()) != 0 {
		T.Fail()
	}

	err = r.Close()
	if err != nil {
		T.Fail()
	}
}
