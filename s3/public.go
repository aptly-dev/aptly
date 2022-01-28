package s3

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/corehandlers"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pkg/errors"
	awsauth "github.com/smira/go-aws-auth"
)

const errCodeNotFound = "NotFound"

// PublishedStorage abstract file system with published files (actually hosted on S3)
type PublishedStorage struct {
	s3               *s3.S3
	config           *aws.Config
	bucket           string
	acl              string
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
func NewPublishedStorageRaw(
	bucket, defaultACL, prefix, storageClass, encryptionMethod string,
	plusWorkaround, disabledMultiDel bool,
	config *aws.Config,
) (*PublishedStorage, error) {
	if defaultACL == "" {
		defaultACL = "private"
	}

	if storageClass == "STANDARD" {
		storageClass = ""
	}

	sess, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}

	result := &PublishedStorage{
		s3:               s3.New(sess),
		bucket:           bucket,
		config:           config,
		acl:              defaultACL,
		prefix:           prefix,
		storageClass:     storageClass,
		encryptionMethod: encryptionMethod,
		plusWorkaround:   plusWorkaround,
		disableMultiDel:  disabledMultiDel,
	}

	return result, nil
}

// NewPublishedStorage creates new instance of PublishedStorage with specified S3 access
// keys, region and bucket name
func NewPublishedStorage(accessKey, secretKey, sessionToken, region, endpoint, bucket, defaultACL, prefix,
	storageClass, encryptionMethod string, plusWorkaround, disableMultiDel, forceSigV2, debug bool) (*PublishedStorage, error) {

	config := &aws.Config{
		Region: aws.String(region),
	}

	if endpoint != "" {
		config = config.WithEndpoint(endpoint).WithS3ForcePathStyle(true)
	}

	if accessKey != "" {
		config.Credentials = credentials.NewStaticCredentials(accessKey, secretKey, sessionToken)
	}

	if debug {
		config = config.WithLogLevel(aws.LogDebug)
	}

	result, err := NewPublishedStorageRaw(bucket, defaultACL, prefix, storageClass,
		encryptionMethod, plusWorkaround, disableMultiDel, config)

	if err == nil && forceSigV2 {
		creds := []awsauth.Credentials{}

		if accessKey != "" {
			creds = append(creds, awsauth.Credentials{
				AccessKeyID:     accessKey,
				SecretAccessKey: secretKey,
			})
		}

		result.s3.Handlers.Sign.Clear()
		result.s3.Handlers.Sign.PushBackNamed(corehandlers.BuildContentLengthHandler)
		result.s3.Handlers.Sign.PushBack(func(req *request.Request) {
			awsauth.SignS3(req.HTTPRequest, creds...)
		})
	}

	return result, err
}

// String
func (storage *PublishedStorage) String() string {
	return fmt.Sprintf("S3: %s:%s/%s", *storage.config.Region, storage.bucket, storage.prefix)
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
	)
	source, err = os.Open(sourceFilename)
	if err != nil {
		return err
	}
	defer source.Close()

	err = storage.putFile(path, source, "")
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("error uploading %s to %s", sourceFilename, storage))
	}

	return err
}

// getMD5 retrieves MD5 stored in the metadata, if any
func (storage *PublishedStorage) getMD5(path string) (string, error) {
	params := &s3.HeadObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(filepath.Join(storage.prefix, path)),
	}
	output, err := storage.s3.HeadObject(params)
	if err != nil {
		return "", err
	}

	return aws.StringValue(output.Metadata["Md5"]), nil
}

// putFile uploads file-like object to
func (storage *PublishedStorage) putFile(path string, source io.ReadSeeker, sourceMD5 string) error {

	params := &s3.PutObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(filepath.Join(storage.prefix, path)),
		Body:   source,
		ACL:    aws.String(storage.acl),
	}
	if storage.storageClass != "" {
		params.StorageClass = aws.String(storage.storageClass)
	}
	if storage.encryptionMethod != "" {
		params.ServerSideEncryption = aws.String(storage.encryptionMethod)
	}
	if sourceMD5 != "" {
		params.Metadata = map[string]*string{
			"Md5": aws.String(sourceMD5),
		}
	}

	_, err := storage.s3.PutObject(params)
	if err != nil {
		return err
	}

	if storage.plusWorkaround && strings.Contains(path, "+") {
		_, err = source.Seek(0, 0)
		if err != nil {
			return err
		}

		return storage.putFile(strings.Replace(path, "+", " ", -1), source, sourceMD5)
	}
	return nil
}

// Remove removes single file under public path
func (storage *PublishedStorage) Remove(path string) error {
	params := &s3.DeleteObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(filepath.Join(storage.prefix, path)),
	}
	_, err := storage.s3.DeleteObject(params)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == s3.ErrCodeNoSuchBucket {
				// ignore 'no such bucket' errors on removal
				return nil
			}
		}
		return errors.Wrap(err, fmt.Sprintf("error deleting %s from %s", path, storage))
	}

	if storage.plusWorkaround && strings.Contains(path, "+") {
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
		if aerr, ok := errors.Cause(err).(awserr.Error); ok {
			if aerr.Code() == s3.ErrCodeNoSuchBucket {
				// ignore 'no such bucket' errors on removal
				return nil
			}
		}
		return err
	}

	if storage.disableMultiDel {
		for i := range filelist {
			params := &s3.DeleteObjectInput{
				Bucket: aws.String(storage.bucket),
				Key:    aws.String(filepath.Join(storage.prefix, path, filelist[i])),
			}
			_, err := storage.s3.DeleteObject(params)
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
			paths := make([]*s3.ObjectIdentifier, len(part))

			for i := range part {
				paths[i] = &s3.ObjectIdentifier{
					Key: aws.String(filepath.Join(storage.prefix, path, part[i])),
				}
			}

			params := &s3.DeleteObjectsInput{
				Bucket: aws.String(storage.bucket),
				Delete: &s3.Delete{
					Objects: paths,
					Quiet:   aws.Bool(true),
				},
			}

			_, err := storage.s3.DeleteObjects(params)
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
func (storage *PublishedStorage) LinkFromPool(publishedDirectory, fileName string, sourcePool aptly.PackagePool,
	sourcePath string, sourceChecksums utils.ChecksumInfo, force bool) error {

	relPath := filepath.Join(publishedDirectory, fileName)
	poolPath := filepath.Join(storage.prefix, relPath)

	if storage.pathCache == nil {
		paths, md5s, err := storage.internalFilelist("", true)
		if err != nil {
			return errors.Wrap(err, "error caching paths under prefix")
		}

		storage.pathCache = make(map[string]string, len(paths))

		for i := range paths {
			storage.pathCache[paths[i]] = md5s[i]
		}
	}

	destinationMD5, exists := storage.pathCache[relPath]
	sourceMD5 := sourceChecksums.MD5

	if exists {
		if len(destinationMD5) != 32 {
			// doesn’t look like a valid MD5,
			// attempt to fetch one from the metadata
			var err error
			destinationMD5, err = storage.getMD5(relPath)
			if err != nil {
				err = errors.Wrap(err, fmt.Sprintf("error verifying MD5 for %s: %s", storage, poolPath))
				return err
			}
			storage.pathCache[relPath] = destinationMD5
		}
		if sourceMD5 == "" {
			return fmt.Errorf("unable to compare object, MD5 checksum missing")
		}

		if destinationMD5 == sourceMD5 {
			return nil
		}

		if !force && destinationMD5 != sourceMD5 {
			return fmt.Errorf("error putting file to %s: file already exists and is different: %s", poolPath, storage)

		}
	}

	source, err := sourcePool.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	err = storage.putFile(relPath, source, sourceMD5)
	if err == nil {
		storage.pathCache[relPath] = sourceMD5
	} else {
		err = errors.Wrap(err, fmt.Sprintf("error uploading %s to %s: %s", sourcePath, storage, poolPath))
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
	prefix = filepath.Join(storage.prefix, prefix)
	if prefix != "" {
		prefix += "/"
	}

	params := &s3.ListObjectsInput{
		Bucket:  aws.String(storage.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: aws.Int64(1000),
	}

	err = storage.s3.ListObjectsPages(params, func(contents *s3.ListObjectsOutput, lastPage bool) bool {
		for _, key := range contents.Contents {
			if storage.plusWorkaround && hidePlusWorkaround && strings.Contains(*key.Key, " ") {
				// if we use plusWorkaround, we want to hide those duplicates
				/// from listing
				continue
			}

			if prefix == "" {
				paths = append(paths, *key.Key)
			} else {
				paths = append(paths, (*key.Key)[len(prefix):])
			}
			md5s = append(md5s, strings.Replace(*key.ETag, "\"", "", -1))
		}

		return true
	})

	if err != nil {
		return nil, nil, errors.WithMessagef(err, "error listing under prefix %s in %s: %s", prefix, storage, err)
	}

	return paths, md5s, nil
}

// RenameFile renames (moves) file
func (storage *PublishedStorage) RenameFile(oldName, newName string) error {
	source := fmt.Sprintf("/%s/%s", storage.bucket, filepath.Join(storage.prefix, oldName))

	params := &s3.CopyObjectInput{
		Bucket:     aws.String(storage.bucket),
		CopySource: aws.String(source),
		Key:        aws.String(filepath.Join(storage.prefix, newName)),
		ACL:        aws.String(storage.acl),
	}

	if storage.storageClass != "" {
		params.StorageClass = aws.String(storage.storageClass)
	}
	if storage.encryptionMethod != "" {
		params.ServerSideEncryption = aws.String(storage.encryptionMethod)
	}

	_, err := storage.s3.CopyObject(params)
	if err != nil {
		return fmt.Errorf("error copying %s -> %s in %s: %s", oldName, newName, storage, err)
	}

	return storage.Remove(oldName)
}

// SymLink creates a copy of src file and adds link information as meta data
func (storage *PublishedStorage) SymLink(src string, dst string) error {

	params := &s3.CopyObjectInput{
		Bucket:     aws.String(storage.bucket),
		CopySource: aws.String(filepath.Join(storage.bucket, storage.prefix, src)),
		Key:        aws.String(filepath.Join(storage.prefix, dst)),
		ACL:        aws.String(storage.acl),
		Metadata: map[string]*string{
			"SymLink": aws.String(src),
		},
		MetadataDirective: aws.String("REPLACE"),
	}

	if storage.storageClass != "" {
		params.StorageClass = aws.String(storage.storageClass)
	}
	if storage.encryptionMethod != "" {
		params.ServerSideEncryption = aws.String(storage.encryptionMethod)
	}

	_, err := storage.s3.CopyObject(params)
	if err != nil {
		return fmt.Errorf("error symlinking %s -> %s in %s: %s", src, dst, storage, err)
	}

	return err
}

// HardLink using symlink functionality as hard links do not exist
func (storage *PublishedStorage) HardLink(src string, dst string) error {
	return storage.SymLink(src, dst)
}

// FileExists returns true if path exists
func (storage *PublishedStorage) FileExists(path string) (bool, error) {
	params := &s3.HeadObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(filepath.Join(storage.prefix, path)),
	}
	_, err := storage.s3.HeadObject(params)
	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == errCodeNotFound {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// ReadLink returns the symbolic link pointed to by path.
// This simply reads text file created with SymLink
func (storage *PublishedStorage) ReadLink(path string) (string, error) {
	params := &s3.HeadObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(filepath.Join(storage.prefix, path)),
	}
	output, err := storage.s3.HeadObject(params)
	if err != nil {
		return "", err
	}

	return aws.StringValue(output.Metadata["SymLink"]), nil
}
