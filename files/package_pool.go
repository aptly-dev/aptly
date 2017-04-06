package files

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/smira/go-uuid/uuid"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
)

// PackagePool is deduplicated storage of package files on filesystem
type PackagePool struct {
	sync.Mutex

	rootPath           string
	supportLegacyPaths bool
}

// Check interface
var (
	_ aptly.PackagePool      = (*PackagePool)(nil)
	_ aptly.LocalPackagePool = (*PackagePool)(nil)
)

// NewPackagePool creates new instance of PackagePool which specified root
func NewPackagePool(root string) *PackagePool {
	rootPath := filepath.Join(root, "pool")
	rootPath, err := filepath.Abs(rootPath)
	if err != nil {
		panic(err)
	}

	return &PackagePool{
		rootPath:           rootPath,
		supportLegacyPaths: true,
	}
}

// LegacyPath returns path relative to pool's root for pre-1.1 aptly (based on MD5)
func (pool *PackagePool) LegacyPath(filename string, checksums *utils.ChecksumInfo) (string, error) {
	filename = filepath.Base(filename)
	if filename == "." || filename == "/" {
		return "", fmt.Errorf("filename %s is invalid", filename)
	}

	hashMD5 := checksums.MD5

	if len(hashMD5) < 4 {
		return "", fmt.Errorf("unable to compute pool location for filename %v, MD5 is missing", filename)
	}

	return filepath.Join(hashMD5[0:2], hashMD5[2:4], filename), nil
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
//
// - srcPath is full path to source file as it is now
// - basename is desired human-readable name (canonical filename)
// - checksums are used to calculate file placement
// - move indicates whether srcPath can be removed
func (pool *PackagePool) Import(srcPath, basename string, checksums *utils.ChecksumInfo, move bool) (string, error) {
	pool.Lock()
	defer pool.Unlock()

	source, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer source.Close()

	sourceInfo, err := source.Stat()
	if err != nil {
		return "", err
	}

	// build target path
	// TODO: replace with new build scheme
	poolPath, err := pool.LegacyPath(basename, checksums)
	if err != nil {
		return "", err
	}

	fullPoolPath := filepath.Join(pool.rootPath, poolPath)

	targetInfo, err := os.Stat(fullPoolPath)
	if err != nil {
		if !os.IsNotExist(err) {
			// unable to stat target location?
			return "", err
		}
	} else {
		// target already exists and same size
		if targetInfo.Size() == sourceInfo.Size() {
			return poolPath, nil
		}

		// trying to overwrite file?
		return "", fmt.Errorf("unable to import into pool: file %s already exists", poolPath)
	}

	if pool.supportLegacyPaths {
		// file doesn't exist at new location, check legacy location
		var (
			legacyTargetInfo           os.FileInfo
			legacyPath, legacyFullPath string
		)

		legacyPath, err = pool.LegacyPath(basename, checksums)
		if err != nil {
			return "", err
		}
		legacyFullPath = filepath.Join(pool.rootPath, legacyPath)

		legacyTargetInfo, err = os.Stat(legacyFullPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return "", err
			}
		} else {
			// legacy file exists
			if legacyTargetInfo.Size() == sourceInfo.Size() {
				// file exists at legacy path and it's same size, consider it's already in the pool
				return legacyPath, nil
			}

			// size is different, import at new path
		}
	}

	// create subdirs as necessary
	poolDir := filepath.Dir(fullPoolPath)
	err = os.MkdirAll(poolDir, 0777)
	if err != nil {
		return "", err
	}

	// check if we can use hardlinks instead of copying/moving
	poolDirInfo, err := os.Stat(poolDir)
	if err != nil {
		return "", err
	}

	if poolDirInfo.Sys().(*syscall.Stat_t).Dev == sourceInfo.Sys().(*syscall.Stat_t).Dev {
		// same filesystem, try to use hardlink
		err = os.Link(srcPath, fullPoolPath)
	} else {
		err = os.ErrInvalid
	}

	if err != nil {
		// different filesystems or failed hardlink, fallback to copy
		var target *os.File
		target, err = os.Create(fullPoolPath)
		if err != nil {
			return "", err
		}
		defer target.Close()

		_, err = io.Copy(target, source)

		if err == nil {
			err = target.Close()
		}
	}

	if err == nil && move {
		err = os.Remove(srcPath)
	}

	return poolPath, err
}

// Open returns io.ReadCloser to access the file
func (pool *PackagePool) Open(path string) (aptly.ReadSeekerCloser, error) {
	return os.Open(filepath.Join(pool.rootPath, path))
}

// Stat returns Unix stat(2) info
func (pool *PackagePool) Stat(path string) (os.FileInfo, error) {
	return os.Stat(filepath.Join(pool.rootPath, path))
}

// Link generates hardlink to destination path
func (pool *PackagePool) Link(path, dstPath string) error {
	return os.Link(filepath.Join(pool.rootPath, path), dstPath)
}

// GenerateTempPath generates temporary path for download (which is fast to import into package pool later on)
func (pool *PackagePool) GenerateTempPath(filename string) (string, error) {
	random := uuid.NewRandom().String()

	return filepath.Join(pool.rootPath, random[0:2], random[2:4], random[4:]+filename), nil
}
