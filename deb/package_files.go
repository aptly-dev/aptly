package deb

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
)

// PackageFile is a single file entry in package
type PackageFile struct {
	// Filename is name of file for the package (without directory)
	Filename string
	// Hashes for the file
	Checksums utils.ChecksumInfo
	// PoolPath persists relative path to file in the package pool
	PoolPath string
	// Temporary field used while downloading, stored relative path on the mirror
	downloadPath string
}

// Verify that package file is present and correct
func (f *PackageFile) Verify(packagePool aptly.PackagePool, checksumStorage aptly.ChecksumStorage) (bool, error) {
	generatedPoolPath, exists, err := packagePool.Verify(f.PoolPath, f.Filename, &f.Checksums, checksumStorage)
	if exists && err == nil {
		f.PoolPath = generatedPoolPath
	}

	return exists, err
}

// GetPoolPath returns path to the file in the pool
//
// For legacy packages which do not have PoolPath field set, that calculates LegacyPath via pool
func (f *PackageFile) GetPoolPath(packagePool aptly.PackagePool) (string, error) {
	var err error

	if f.PoolPath == "" {
		f.PoolPath, err = packagePool.LegacyPath(f.Filename, &f.Checksums)
	}

	return f.PoolPath, err
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

func (files PackageFiles) parseSumField(input string, setter func(sum *utils.ChecksumInfo, data string)) (PackageFiles, error) {
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)

		if len(parts) < 3 {
			return nil, fmt.Errorf("unparseable hash sum line: %#v", line)
		}

		size, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("unable to parse size: %s", err)
		}

		filename := filepath.Base(parts[len(parts)-1])

		found := false
		pos := 0
		for i, file := range files {
			if file.Filename == filename {
				found = true
				pos = i
				break
			}
		}

		if !found {
			files = append(files, PackageFile{Filename: filename})
			pos = len(files) - 1
		}

		files[pos].Checksums.Size = size
		setter(&files[pos].Checksums, parts[0])
	}

	return files, nil
}

// ParseSumFields populates PackageFiles by parsing stanza checksums fields
func (files PackageFiles) ParseSumFields(stanza Stanza) (PackageFiles, error) {
	var err error

	files, err = files.parseSumField(stanza["Files"], func(sum *utils.ChecksumInfo, data string) { sum.MD5 = data })
	if err != nil {
		return nil, err
	}

	files, err = files.parseSumField(stanza["Checksums-Sha1"], func(sum *utils.ChecksumInfo, data string) { sum.SHA1 = data })
	if err != nil {
		return nil, err
	}

	files, err = files.parseSumField(stanza["Checksums-Sha256"], func(sum *utils.ChecksumInfo, data string) { sum.SHA256 = data })
	if err != nil {
		return nil, err
	}

	files, err = files.parseSumField(stanza["Checksums-Sha512"], func(sum *utils.ChecksumInfo, data string) { sum.SHA512 = data })
	if err != nil {
		return nil, err
	}

	return files, nil
}
