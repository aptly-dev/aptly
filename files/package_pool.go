package files

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/smira/aptly/aptly"
)

// PackagePool is deduplicated storage of package files on filesystem
type PackagePool struct {
	sync.Mutex
	rootPath string
	hashSelector string
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
func (pool *PackagePool) RelativePath(filename string, hash string) (string, error) {
	filename = filepath.Base(filename)
	if filename == "." || filename == "/" {
		return "", fmt.Errorf("filename %s is invalid", filename)
	}

	if len(hash) < 4 {
		return "", fmt.Errorf("unable to compute pool location for filename %v, %s is missing", filename, pool.HashSelector())
	}

	return filepath.Join(hash[0:2], hash[2:4], filename), nil
}

// Return name of hash that is used for selecting the file path
func (pool *PackagePool) HashSelector() string {
	if pool.hashSelector == "" {
		pool.hashSelector = "MD5"
	} 

	return pool.hashSelector
}

// Set the name of hash selector that is used for selecting the file path
func (pool *PackagePool) SetHashSelector(hashSelector string) {
	if hashSelector == "MD5" || hashSelector == "md5" || hashSelector == "MD5Sum" || hashSelector == "MD5sum" {
		hashSelector = "MD5"
	} else if hashSelector == "SHA1" || hashSelector == "sha1" {
		hashSelector = "SHA1"
	} else if hashSelector == "SHA256" || hashSelector == "sha256" {
		hashSelector = "SHA256"
	} else if hashSelector == "SHA512" || hashSelector == "sha512" {
		hashSelector = "SHA512"
	} else {
		if hashSelector != "" {
			fmt.Printf("Invalid hash name %s, defaulting to MD5\n", hashSelector)
		}
		hashSelector = "MD5"
	}

	// You only have one chance to set the hash selector
	if pool.hashSelector == "" {
		pool.hashSelector = hashSelector
	} else if pool.hashSelector != hashSelector {
		fmt.Printf("Hash name used for file paths can only be set once, current %s, new %s\n", pool.hashSelector, hashSelector)
	}
}

// Path returns full path to package file in pool given any name and hash of file contents
func (pool *PackagePool) Path(filename string, hash string) (string, error) {
	relative, err := pool.RelativePath(filename, hash)
	if err != nil {
		return "", err
	}

	return filepath.Join(pool.rootPath, relative), nil
}

// FilepathList returns file paths of all the files in the pool
func (pool *PackagePool) FilepathList(progress aptly.Progress) ([]string, error) {
	pool.Lock()
	defer pool.Unlock()

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
	pool.Lock()
	defer pool.Unlock()

	path = filepath.Join(pool.rootPath, path)

	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}

	err = os.Remove(path)
	return info.Size(), err
}

// Import copies file into package pool
func (pool *PackagePool) Import(path string, hash string) error {
	pool.Lock()
	defer pool.Unlock()

	source, err := os.Open(path)
	if err != nil {
		return err
	}
	defer source.Close()

	sourceInfo, err := source.Stat()
	if err != nil {
		return err
	}

	poolPath, err := pool.Path(path, hash)
	if err != nil {
		return err
	}

	targetInfo, err := os.Stat(poolPath)
	if err != nil {
		if !os.IsNotExist(err) {
			// unable to stat target location?
			return err
		}
	} else {
		// target already exists
		if targetInfo.Size() != sourceInfo.Size() {
			// trying to overwrite file?
			return fmt.Errorf("unable to import into pool: file %s already exists", poolPath)
		}

		// assume that target is already there
		return nil
	}

	// create subdirs as necessary
	err = os.MkdirAll(filepath.Dir(poolPath), 0777)
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
