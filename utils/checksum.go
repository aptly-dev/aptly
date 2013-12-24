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
	result := &ChecksumInfo{}

	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	result.Size = st.Size()

	hashes := []hash.Hash{md5.New(), sha1.New(), sha256.New()}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := make([]byte, 8192)
	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		for _, h := range hashes {
			h.Write(buf[:n])
		}
	}

	result.MD5 = fmt.Sprintf("%x", hashes[0].Sum(nil))
	result.SHA1 = fmt.Sprintf("%x", hashes[1].Sum(nil))
	result.SHA256 = fmt.Sprintf("%x", hashes[2].Sum(nil))

	return result, nil
}
