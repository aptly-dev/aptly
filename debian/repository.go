package debian

import (
	"fmt"
	"github.com/smira/aptly/utils"
	"os"
	"path/filepath"
	"strings"
)

// Repository directory structure:
// <root>
// \- pool
//    \- ab
//       \- ae
//          \- package.deb
// \- public
//    \- dists
//       \- squeeze
//          \- Release
//          \- main
//             \- binary-i386
//                \- Packages.bz2
//                   references packages from pool
//    \- pool
//       contains symlinks to main pool

// Repository abstract file system with package pool and published package repos
type Repository struct {
	RootPath string
}

// NewRepository creates new instance of repository which specified root
func NewRepository(root string) *Repository {
	return &Repository{RootPath: root}
}

// PoolPath returns full path to package file in pool givan any name and hash of file contents
func (r *Repository) PoolPath(filename string, hashMD5 string) (string, error) {
	filename = filepath.Base(filename)
	if filename == "." || filename == "/" {
		return "", fmt.Errorf("filename %s is invalid", filename)
	}

	return filepath.Join(r.RootPath, "pool", hashMD5[0:2], hashMD5[2:4], filename), nil
}

// MkDir creates directory recursively under public path
func (r *Repository) MkDir(path string) error {
	return os.MkdirAll(filepath.Join(r.RootPath, "public", path), 0755)
}

// CreateFile creates file for writing under public path
func (r *Repository) CreateFile(path string) (*os.File, error) {
	return os.Create(filepath.Join(r.RootPath, "public", path))
}

// LinkFromPool links package file from pool to dist's pool location
func (r *Repository) LinkFromPool(prefix string, component string, filename string, hashMD5 string) error {
	sourcePath, err := r.PoolPath(filename, hashMD5)
	if err != nil {
		return err
	}

	baseName := filepath.Base(filename)

	if len(baseName) < 8 {
		return fmt.Errorf("package filename %s too short", baseName)
	}

	var subdir string
	if strings.HasPrefix(baseName, "lib") {
		subdir = baseName[:4]
	} else {
		subdir = baseName[:1]

	}

	poolPath := filepath.Join(r.RootPath, "public", prefix, "pool", component, subdir)

	err = os.MkdirAll(poolPath, 0755)
	if err != nil {
		return err
	}

	err = os.Link(sourcePath, filepath.Join(poolPath, baseName))
	return err
}

// ChecksumsForFile proxies requests to utils.ChecksumsForFile, joining public path
func (r *Repository) ChecksumsForFile(path string) (*utils.ChecksumInfo, error) {
	return utils.ChecksumsForFile(filepath.Join(r.RootPath, "public", path))
}
