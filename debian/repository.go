package debian

import (
	"fmt"
	"os"
	"path/filepath"
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
