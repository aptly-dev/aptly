package s3

import (
	"fmt"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/files"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PublishedStorage abstract file system with published files (actually hosted on S3)
type PublishedStorage struct {
	s3               *s3.S3
	bucket           *s3.Bucket
	acl              s3.ACL
	prefix           string
	storageClass     string
	encryptionMethod string
	plusWorkaround   bool
	disableMultiDel  bool
	pathCache        map[string]string
}

// Check interface
var (
	_ aptly.PublishedStorage = (*PublishedStorage)(nil)
)

// NewPublishedStorageRaw creates published storage from raw aws credentials
func NewPublishedStorageRaw(auth aws.Auth, region aws.Region, bucket, defaultACL, prefix,
	storageClass, encryptionMethod string, plusWorkaround, disabledMultiDel bool) (*PublishedStorage, error) {
	if defaultACL == "" {
		defaultACL = "private"
	}

	if storageClass == "STANDARD" {
		storageClass = ""
	}

	result := &PublishedStorage{
		s3:               s3.New(auth, region),
		acl:              s3.ACL(defaultACL),
		prefix:           prefix,
		storageClass:     storageClass,
		encryptionMethod: encryptionMethod,
		plusWorkaround:   plusWorkaround,
		disableMultiDel:  disabledMultiDel,
	}

	result.s3.HTTPClient = func() *http.Client {
		return RetryingClient
	}
	result.bucket = result.s3.Bucket(bucket)

	return result, nil
}

// NewPublishedStorage creates new instance of PublishedStorage with specified S3 access
// keys, region and bucket name
func NewPublishedStorage(accessKey, secretKey, region, endpoint, bucket, defaultACL, prefix,
	storageClass, encryptionMethod string, plusWorkaround, disableMultiDel bool) (*PublishedStorage, error) {
	auth, err := aws.GetAuth(accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	var awsRegion aws.Region

	if endpoint == "" {
		var ok bool

		awsRegion, ok = aws.Regions[region]
		if !ok {
			return nil, fmt.Errorf("unknown region: %#v", region)
		}
	} else {
		awsRegion = aws.Region{
			Name:                 region,
			S3Endpoint:           endpoint,
			S3LocationConstraint: true,
			S3LowercaseBucket:    true,
		}
	}

	return NewPublishedStorageRaw(auth, awsRegion, bucket, defaultACL, prefix, storageClass, encryptionMethod,
		plusWorkaround, disableMultiDel)
}

// String
func (storage *PublishedStorage) String() string {
	return fmt.Sprintf("S3: %s:%s/%s", storage.s3.Region.Name, storage.bucket.Name, storage.prefix)
}

// MkDir creates directory recursively under public path
func (storage *PublishedStorage) MkDir(path string) error {
	// no op for S3
	return nil
}

// PutFile puts file into published storage at specified path
func (storage *PublishedStorage) PutFile(path string, sourceFilename string) error {
	var (
		source *os.File
		err    error
		fi     os.FileInfo
	)
	source, err = os.Open(sourceFilename)
	if err != nil {
		return err
	}
	defer source.Close()

	fi, err = source.Stat()
	if err != nil {
		return err
	}

	headers := map[string][]string{
		"Content-Type": {"binary/octet-stream"},
	}
	if storage.storageClass != "" {
		headers["x-amz-storage-class"] = []string{storage.storageClass}
	}
	if storage.encryptionMethod != "" {
		headers["x-amz-server-side-encryption"] = []string{storage.encryptionMethod}
	}

	err = storage.bucket.PutReaderHeader(filepath.Join(storage.prefix, path), source, fi.Size(), headers, storage.acl)
	if err != nil {
		return fmt.Errorf("error uploading %s to %s: %s", sourceFilename, storage, err)
	}

	if storage.plusWorkaround && strings.Index(path, "+") != -1 {
		return storage.PutFile(strings.Replace(path, "+", " ", -1), sourceFilename)
	}
	return nil
}

// Remove removes single file under public path
func (storage *PublishedStorage) Remove(path string) error {
	err := storage.bucket.Del(filepath.Join(storage.prefix, path))
	if err != nil {
		return fmt.Errorf("error deleting %s from %s: %s", path, storage, err)
	}

	if storage.plusWorkaround && strings.Index(path, "+") != -1 {
		// try to remove workaround version, but don't care about result
		_ = storage.Remove(strings.Replace(path, "+", " ", -1))
	}
	return nil
}

// RemoveDirs removes directory structure under public path
func (storage *PublishedStorage) RemoveDirs(path string, progress aptly.Progress) error {
	const page = 1000

	filelist, _, err := storage.internalFilelist(path, false)
	if err != nil {
		return err
	}

	if storage.disableMultiDel {
		for i := range filelist {
			err = storage.bucket.Del(filepath.Join(storage.prefix, path, filelist[i]))
			if err != nil {
				return fmt.Errorf("error deleting path %s from %s: %s", filelist[i], storage, err)
			}
		}
	} else {
		numParts := (len(filelist) + page - 1) / page

		for i := 0; i < numParts; i++ {
			var part []string
			if i == numParts-1 {
				part = filelist[i*page:]
			} else {
				part = filelist[i*page : (i+1)*page]
			}
			paths := make([]string, len(part))

			for i := range part {
				paths[i] = filepath.Join(storage.prefix, path, part[i])
			}

			err = storage.bucket.MultiDel(paths)
			if err != nil {
				return fmt.Errorf("error deleting multiple paths from %s: %s", storage, err)
			}
		}
	}

	return nil
}

// LinkFromPool links package file from pool to dist's pool location
//
// publishedDirectory is desired location in pool (like prefix/pool/component/liba/libav/)
// sourcePool is instance of aptly.PackagePool
// sourcePath is filepath to package file in package pool
//
// LinkFromPool returns relative path for the published file to be included in package index
func (storage *PublishedStorage) LinkFromPool(publishedDirectory string, sourcePool aptly.PackagePool,
	sourcePath, sourceMD5 string, force bool) error {
	// verify that package pool is local pool in filesystem
	_ = sourcePool.(*files.PackagePool)

	baseName := filepath.Base(sourcePath)
	relPath := filepath.Join(publishedDirectory, baseName)
	poolPath := filepath.Join(storage.prefix, relPath)

	var (
		err error
	)

	if storage.pathCache == nil {
		paths, md5s, err := storage.internalFilelist(storage.prefix, true)
		if err != nil {
			return fmt.Errorf("error caching paths under prefix: %s", err)
		}

		storage.pathCache = make(map[string]string, len(paths))

		for i := range paths {
			storage.pathCache[paths[i]] = md5s[i]
		}
	}

	destinationMD5, exists := storage.pathCache[relPath]

	if exists {
		if destinationMD5 == sourceMD5 {
			return nil
		}

		if !force && destinationMD5 != sourceMD5 {
			return fmt.Errorf("error putting file to %s: file already exists and is different: %s", poolPath, storage)

		}
	}

	err = storage.PutFile(relPath, sourcePath)
	if err == nil {
		storage.pathCache[relPath] = sourceMD5
	}

	return err
}

// Filelist returns list of files under prefix
func (storage *PublishedStorage) Filelist(prefix string) ([]string, error) {
	paths, _, err := storage.internalFilelist(prefix, true)
	return paths, err
}

func (storage *PublishedStorage) internalFilelist(prefix string, hidePlusWorkaround bool) (paths []string, md5s []string, err error) {
	paths = make([]string, 0, 1024)
	md5s = make([]string, 0, 1024)
	marker := ""
	prefix = filepath.Join(storage.prefix, prefix)
	if prefix != "" {
		prefix += "/"
	}
	for {
		contents, err := storage.bucket.List(prefix, "", marker, 1000)
		if err != nil {
			return nil, nil, fmt.Errorf("error listing under prefix %s in %s: %s", prefix, storage, err)
		}
		lastKey := ""
		for _, key := range contents.Contents {
			lastKey = key.Key
			if storage.plusWorkaround && hidePlusWorkaround && strings.Index(lastKey, " ") != -1 {
				// if we use plusWorkaround, we want to hide those duplicates
				/// from listing
				continue
			}

			if prefix == "" {
				paths = append(paths, key.Key)
			} else {
				paths = append(paths, key.Key[len(prefix):])
			}
			md5s = append(md5s, strings.Replace(key.ETag, "\"", "", -1))
		}
		if contents.IsTruncated {
			marker = contents.NextMarker
			if marker == "" {
				// From the s3 docs: If response does not include the
				// NextMarker and it is truncated, you can use the value of the
				// last Key in the response as the marker in the subsequent
				// request to get the next set of object keys.
				marker = lastKey
			}
		} else {
			break
		}
	}

	return paths, md5s, nil
}

// RenameFile renames (moves) file
func (storage *PublishedStorage) RenameFile(oldName, newName string) error {
	err := storage.bucket.Copy(filepath.Join(storage.prefix, oldName), filepath.Join(storage.prefix, newName), storage.acl)
	if err != nil {
		return fmt.Errorf("error copying %s -> %s in %s: %s", oldName, newName, storage, err)
	}

	return storage.Remove(oldName)
}
