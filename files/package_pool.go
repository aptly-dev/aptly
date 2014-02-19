package files

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
	"io/ioutil"
	"os"
	"path/filepath"
)

// PackagePool is deduplicated storage of package files on filesystem
type PackagePool struct {
	rootPath string
}

// Check interface
var (
	_ aptly.PackagePool = (*PackagePool)(nil)
)

// NewPackagePool creates new instance of PackagePool which specified root
func NewPackagePool(root string) *PackagePool {
	return &PackagePool{rootPath: filepath.Join(root, "pool")}
}

// RelativePath returns path relative to pool's root for package files given MD5 and original filename
func (pool *PackagePool) RelativePath(filename string, hashMD5 string) (string, error) {
	filename = filepath.Base(filename)
	if filename == "." || filename == "/" {
		return "", fmt.Errorf("filename %s is invalid", filename)
	}

	if len(hashMD5) < 4 {
		return "", fmt.Errorf("unable to compute pool location for filename %v, MD5 is missing", filename)
	}

	return filepath.Join(hashMD5[0:2], hashMD5[2:4], filename), nil
}

// Path returns full path to package file in pool given any name and hash of file contents
func (p *PackagePool) Path(filename string, hashMD5 string) (string, error) {
	relative, err := p.RelativePath(filename, hashMD5)
	if err != nil {
		return "", err
	}

	return filepath.Join(p.rootPath, relative), nil
}

// FilepathList returns file paths of all the files in the pool
func (p *PackagePool) FilepathList(progress *utils.Progress) ([]string, error) {
	dirs, err := ioutil.ReadDir(p.rootPath)
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
		err = filepath.Walk(filepath.Join(p.rootPath, dir.Name()), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				result = append(result, path[len(p.rootPath)+1:])
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

// Remove deletes file in package pool returns its size
func (p *PackagePool) Remove(path string) (size int64, err error) {
	path = filepath.Join(p.rootPath, path)

	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	err = os.Remove(path)
	return info.Size(), err
}
