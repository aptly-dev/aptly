package debian

import (
	"fmt"
	"github.com/smira/aptly/utils"
	"io/ioutil"
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

// RelativePoolPath returns path relative to pool's root
func (r *Repository) RelativePoolPath(filename string, hashMD5 string) (string, error) {
	filename = filepath.Base(filename)
	if filename == "." || filename == "/" {
		return "", fmt.Errorf("filename %s is invalid", filename)
	}

	if len(hashMD5) < 4 {
		return "", fmt.Errorf("unable to compute pool location for filename %v, MD5 is missing", filename)
	}

	return filepath.Join(hashMD5[0:2], hashMD5[2:4], filename), nil
}

// PoolPath returns full path to package file in pool given any name and hash of file contents
func (r *Repository) PoolPath(filename string, hashMD5 string) (string, error) {
	relative, err := r.RelativePoolPath(filename, hashMD5)
	if err != nil {
		return "", err
	}

	return filepath.Join(r.RootPath, "pool", relative), nil
}

// PublicPath returns root of public part of repository
func (r *Repository) PublicPath() string {
	return filepath.Join(r.RootPath, "public")
}

// MkDir creates directory recursively under public path
func (r *Repository) MkDir(path string) error {
	return os.MkdirAll(filepath.Join(r.RootPath, "public", path), 0755)
}

// CreateFile creates file for writing under public path
func (r *Repository) CreateFile(path string) (*os.File, error) {
	return os.Create(filepath.Join(r.RootPath, "public", path))
}

// RemoveDirs removes directory structure under public path
func (r *Repository) RemoveDirs(path string) error {
	filepath := filepath.Join(r.RootPath, "public", path)
	fmt.Printf("Removing %s...\n", filepath)
	return os.RemoveAll(filepath)
}

// LinkFromPool links package file from pool to dist's pool location
func (r *Repository) LinkFromPool(prefix string, component string, sourcePath string, poolDirectory string) (string, error) {
	baseName := filepath.Base(sourcePath)

	relPath := filepath.Join("pool", component, poolDirectory, baseName)
	poolPath := filepath.Join(r.RootPath, "public", prefix, "pool", component, poolDirectory)

	err := os.MkdirAll(poolPath, 0755)
	if err != nil {
		return "", err
	}

	_, err = os.Stat(filepath.Join(poolPath, baseName))
	if err == nil { // already exists, skip
		return relPath, nil
	}

	err = os.Link(sourcePath, filepath.Join(poolPath, baseName))
	return relPath, err
}

// ChecksumsForFile proxies requests to utils.ChecksumsForFile, joining public path
func (r *Repository) ChecksumsForFile(path string) (utils.ChecksumInfo, error) {
	return utils.ChecksumsForFile(filepath.Join(r.RootPath, "public", path))
}

// PoolFilepathList returns file paths of all the files in the pool
func (r *Repository) PoolFilepathList(progress *utils.Progress) ([]string, error) {
	poolPath := filepath.Join(r.RootPath, "pool")

	dirs, err := ioutil.ReadDir(poolPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	if len(dirs) == 0 {
		return nil, nil
	}

	if progress != nil {
		progress.InitBar(int64(len(dirs)), false)
		defer progress.ShutdownBar()
	}

	result := []string{}

	for _, dir := range dirs {
		err = filepath.Walk(filepath.Join(poolPath, dir.Name()), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				result = append(result, path[len(poolPath)+1:])
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		if progress != nil {
			progress.AddBar(1)
		}
	}

	return result, nil
}
