package s3

import (
	"fmt"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/files"
	"os"
	"path/filepath"
)

// PublishedStorage abstract file system with published files (actually hosted on S3)
type PublishedStorage struct {
	s3     *s3.S3
	bucket *s3.Bucket
	acl    s3.ACL
	prefix string
}

// Check interface
var (
	_ aptly.PublishedStorage = (*PublishedStorage)(nil)
)

func NewPublishedStorageRaw(auth aws.Auth, region aws.Region, bucket, defaultACL, prefix string) (*PublishedStorage, error) {
	if defaultACL == "" {
		defaultACL = "private"
	}

	result := &PublishedStorage{s3: s3.New(auth, region), acl: s3.ACL(defaultACL), prefix: prefix}
	result.bucket = result.s3.Bucket(bucket)

	return result, nil
}

// NewPublishedStorage creates new instance of PublishedStorage with specified S3 access
// keys, region and bucket name
func NewPublishedStorage(accessKey, secretKey, region, bucket, defaultACL, prefix string) (*PublishedStorage, error) {
	auth, err := aws.GetAuth(accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	awsRegion, ok := aws.Regions[region]
	if !ok {
		return nil, fmt.Errorf("unknown region: %#v", region)
	}

	return NewPublishedStorageRaw(auth, awsRegion, bucket, defaultACL, prefix)
}

// PublicPath returns root of public part
func (storage *PublishedStorage) PublicPath() string {
	panic("never would be implemented")
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

	return storage.bucket.PutReader(filepath.Join(storage.prefix, path), source, fi.Size(), "binary/octet-stream", storage.acl)
}

// Remove removes single file under public path
func (storage *PublishedStorage) Remove(path string) error {
	return storage.bucket.Del(filepath.Join(storage.prefix, path))
}

// RemoveDirs removes directory structure under public path
func (storage *PublishedStorage) RemoveDirs(path string, progress aptly.Progress) error {
	const page = 1000

	filelist, err := storage.Filelist(path)
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
			return err
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
func (storage *PublishedStorage) LinkFromPool(publishedDirectory string, sourcePool aptly.PackagePool, sourcePath, sourceMD5 string) error {
	// verify that package pool is local pool in filesystem
	_ = sourcePool.(*files.PackagePool)

	baseName := filepath.Base(sourcePath)
	relPath := filepath.Join(publishedDirectory, baseName)
	poolPath := filepath.Join(storage.prefix, relPath)

	var (
		dstKey *s3.Key
		err    error
	)

	dstKey, err = storage.bucket.GetKey(poolPath)
	if err != nil {
		if s3err, ok := err.(*s3.Error); !ok || s3err.StatusCode != 404 {
			return err
		}
	} else {
		if dstKey.ETag == sourceMD5 {
			return nil
		}
	}

	return storage.PutFile(relPath, sourcePath)
}

// Filelist returns list of files under prefix
func (storage *PublishedStorage) Filelist(prefix string) ([]string, error) {
	result := []string{}
	marker := ""
	prefix = filepath.Join(storage.prefix, prefix)
	if prefix != "" {
		prefix += "/"
	}
	for {
		contents, err := storage.bucket.List(prefix, "", marker, 1000)
		if err != nil {
			return nil, err
		}
		last_key := ""
		for _, key := range contents.Contents {
			if prefix == "" {
				result = append(result, key.Key)
			} else {
				result = append(result, key.Key[len(prefix):])
			}
			last_key = key.Key
		}
		if contents.IsTruncated {
			marker = contents.NextMarker
			if marker == "" {
				// From the s3 docs: If response does not include the
				// NextMarker and it is truncated, you can use the value of the
				// last Key in the response as the marker in the subsequent
				// request to get the next set of object keys.
				marker = last_key
			}
		} else {
			break
		}
	}

	return result, nil
}

// RenameFile renames (moves) file
func (storage *PublishedStorage) RenameFile(oldName, newName string) error {
	panic("not implemented yet")
}
