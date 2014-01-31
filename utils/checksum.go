package utils

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
)

// ChecksumInfo represents checksums for a single file
type ChecksumInfo struct {
	Size   int64
	MD5    string
	SHA1   string
	SHA256 string
}

// ChecksumsForFile generates size, MD5, SHA1 & SHA256 checksums for given file
func ChecksumsForFile(path string) (*ChecksumInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	w := NewChecksumWriter(&DevNull{})

	_, err = io.Copy(w, file)
	if err != nil {
		return nil, err
	}

	return w.Sum(), nil
}

// DevNull just accepts anything
type DevNull struct {
}

// Write to DevNull always succeeds
func (d *DevNull) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// ChecksumWriter is a writer that does checksum calculation on the fly passing data
// to real writer
type ChecksumWriter struct {
	w      io.Writer
	sum    ChecksumInfo
	hashes []hash.Hash
}

// Interface check
var (
	_ io.Writer = &ChecksumWriter{}
)

// NewChecksumWriter creates checksum calculator for given writer w
func NewChecksumWriter(w io.Writer) *ChecksumWriter {
	return &ChecksumWriter{
		w:      w,
		hashes: []hash.Hash{md5.New(), sha1.New(), sha256.New()},
	}
}

// Write implememnts pass-through writing with checksum calculation on the fly
func (c *ChecksumWriter) Write(p []byte) (n int, err error) {
	n, err = c.w.Write(p)
	if err != nil {
		return n, err
	}

	c.sum.Size += int64(n)

	for _, h := range c.hashes {
		h.Write(p[:n])
	}

	return
}

// Sum returns caculated ChecksumInfo
func (c *ChecksumWriter) Sum() *ChecksumInfo {
	c.sum.MD5 = fmt.Sprintf("%x", c.hashes[0].Sum(nil))
	c.sum.SHA1 = fmt.Sprintf("%x", c.hashes[1].Sum(nil))
	c.sum.SHA256 = fmt.Sprintf("%x", c.hashes[2].Sum(nil))

	return &c.sum
}
