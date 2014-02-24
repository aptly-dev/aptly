package files

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"io"
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
func (pool *PackagePool) Path(filename string, hashMD5 string) (string, error) {
	relative, err := pool.RelativePath(filename, hashMD5)
	if err != nil {
		return "", err
	}

	return filepath.Join(pool.rootPath, relative), nil
}

// FilepathList returns file paths of all the files in the pool
func (pool *PackagePool) FilepathList(progress aptly.Progress) ([]string, error) {
	dirs, err := ioutil.ReadDir(pool.rootPath)
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
		err = filepath.Walk(filepath.Join(pool.rootPath, dir.Name()), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				result = append(result, path[len(pool.rootPath)+1:])
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
func (pool *PackagePool) Remove(path string) (size int64, err error) {
	path = filepath.Join(pool.rootPath, path)

	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	err = os.Remove(path)
	return info.Size(), err
}

// Import copies file into package pool
func (pool *PackagePool) Import(path string, hashMD5 string) error {
	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()

	sourceInfo, err := source.Stat()
	if err != nil {
		return err
	}

	poolPath, err := pool.Path(path, hashMD5)
	if err != nil {
		return err
	}

	targetInfo, err := os.Stat(poolPath)
	if err != nil {
		if !os.IsNotExist(err) {
			// unable to stat target location?
			return err
		}
		// file doesn't exist, that's ok
	} else {
		if targetInfo.Size() != sourceInfo.Size() {
			// trying to overwrite file?
			return fmt.Errorf("unable to import into pool: file %s already exists", poolPath)
		}
	}

	// create subdirs as necessary
	err = os.MkdirAll(filepath.Dir(poolPath), 0755)
	if err != nil {
		return err
	}

	target, err := os.Create(poolPath)
	if err != nil {
		return err
	}
	defer target.Close()

	_, err = io.Copy(target, source)

	return err
}
