package deb

import (
	"encoding/binary"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
)

// PackageFile is a single file entry in package
type PackageFile struct {
	// Filename is name of file for the package (without directory)
	Filename string
	// Hashes for the file
	Checksums utils.ChecksumInfo
	// Temporary field used while downloading, stored relative path on the mirror
	downloadPath string
}

// Verify that package file is present and correct
func (f *PackageFile) Verify(packagePool aptly.PackagePool) (bool, error) {
	poolPath, err := packagePool.Path(f.Filename, f.Checksums.MD5)
	if err != nil {
		return false, err
	}

	st, err := os.Stat(poolPath)
	if err != nil {
		return false, nil
	}

	// verify size
	// TODO: verify checksum if configured
	return st.Size() == f.Checksums.Size, nil
}

// DownloadURL return relative URL to package download location
func (f *PackageFile) DownloadURL() string {
	return filepath.Join(f.downloadPath, f.Filename)
}

// PackageFiles is collection of package files
type PackageFiles []PackageFile

// Hash compute hash of all file items, sorting them first
func (files PackageFiles) Hash() uint64 {
	sort.Sort(files)

	h := fnv.New64a()

	for _, f := range files {
		h.Write([]byte(f.Filename))
		binary.Write(h, binary.BigEndian, f.Checksums.Size)
		h.Write([]byte(f.Checksums.MD5))
		h.Write([]byte(f.Checksums.SHA1))
		h.Write([]byte(f.Checksums.SHA256))
	}

	return h.Sum64()
}

// Len returns number of files
func (files PackageFiles) Len() int {
	return len(files)
}

// Swap swaps elements
func (files PackageFiles) Swap(i, j int) {
	files[i], files[j] = files[j], files[i]
}

// Less compares by filename
func (files PackageFiles) Less(i, j int) bool {
	return files[i].Filename < files[j].Filename
}
