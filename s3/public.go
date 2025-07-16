package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go"
	"github.com/aws/smithy-go/logging"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type logger struct{}

func (l *logger) Logf(classification logging.Classification, format string, v ...interface{}) {
	var e *zerolog.Event
	switch classification {
	case logging.Debug:
		e = log.Logger.Debug()
	case logging.Warn:
		e = log.Logger.Warn()
	default:
		e = log.Logger.Error()
	}
	e.Msgf(format, v...)
}

// PublishedStorage abstract file system with published files (actually hosted on S3)
type PublishedStorage struct {
	s3               *s3.Client
	config           *aws.Config
	bucket           string
	acl              types.ObjectCannedACL
	prefix           string
	storageClass     types.StorageClass
	encryptionMethod types.ServerSideEncryption
	plusWorkaround   bool
	disableMultiDel  bool
	pathCache        map[string]string
	pathCacheMutex   sync.RWMutex

	// True if the bucket encrypts objects by default.
	encryptByDefault bool

	// Concurrent upload configuration
	concurrentUploads int
	uploadQueue       chan *uploadTask
	uploadErrors      chan error
	uploadWg          sync.WaitGroup
}

// uploadTask represents a file upload job
type uploadTask struct {
	path           string
	sourceFilename string
	sourceReader   io.ReadSeeker
	sourceMD5      string
	isFile         bool // true for PutFile, false for putFile with reader
}

// Check interface
var (
	_ aptly.PublishedStorage = (*PublishedStorage)(nil)
)

// NewPublishedStorageRaw creates published storage from raw aws credentials
func NewPublishedStorageRaw(
	bucket, defaultACL, prefix, storageClass, encryptionMethod string,
	plusWorkaround, disabledMultiDel, forceVirtualHostedStyle bool,
	config *aws.Config, endpoint string, concurrentUploads int, uploadQueueSize int,
) (*PublishedStorage, error) {
	var acl types.ObjectCannedACL
	if defaultACL == "" || defaultACL == "private" {
		acl = types.ObjectCannedACLPrivate
	} else if defaultACL == "public-read" {
		acl = types.ObjectCannedACLPublicRead
	} else if defaultACL == "none" {
		acl = ""
	}

	if storageClass == string(types.StorageClassStandard) {
		storageClass = ""
	}

	var baseEndpoint *string
	if endpoint != "" {
		baseEndpoint = aws.String(endpoint)
	}

	result := &PublishedStorage{
		s3: s3.NewFromConfig(*config, func(o *s3.Options) {
			o.UsePathStyle = !forceVirtualHostedStyle
			o.HTTPSignerV4 = signer.NewSigner()
			o.BaseEndpoint = baseEndpoint
		}),
		bucket:            bucket,
		config:            config,
		acl:               acl,
		prefix:            prefix,
		storageClass:      types.StorageClass(storageClass),
		encryptionMethod:  types.ServerSideEncryption(encryptionMethod),
		plusWorkaround:    plusWorkaround,
		disableMultiDel:   disabledMultiDel,
		concurrentUploads: concurrentUploads,
	}

	// Initialize concurrent upload infrastructure if enabled
	if concurrentUploads > 0 {
		// Default queue size is 2x the number of workers if not specified
		if uploadQueueSize <= 0 {
			uploadQueueSize = 2
		}
		queueSize := concurrentUploads * uploadQueueSize

		result.uploadQueue = make(chan *uploadTask, queueSize)
		result.uploadErrors = make(chan error, 1)

		// Start upload workers
		for i := 0; i < concurrentUploads; i++ {
			go result.uploadWorker()
		}
	}

	result.setKMSFlag()

	return result, nil
}

func (storage *PublishedStorage) setKMSFlag() {
	params := &s3.GetBucketEncryptionInput{
		Bucket: aws.String(storage.bucket),
	}
	output, err := storage.s3.GetBucketEncryption(context.TODO(), params)
	if err != nil {
		return
	}

	if len(output.ServerSideEncryptionConfiguration.Rules) > 0 &&
		output.ServerSideEncryptionConfiguration.Rules[0].ApplyServerSideEncryptionByDefault.SSEAlgorithm == "aws:kms" {
		storage.encryptByDefault = true
	}
}

// uploadWorker processes upload tasks from the queue
func (storage *PublishedStorage) uploadWorker() {
	for task := range storage.uploadQueue {
		var err error

		if task.isFile {
			// Handle file upload
			source, openErr := os.Open(task.sourceFilename)
			if openErr != nil {
				err = errors.Wrap(openErr, fmt.Sprintf("error opening %s", task.sourceFilename))
			} else {
				err = storage.putFile(task.path, source, task.sourceMD5)
				_ = source.Close()
				if err != nil {
					err = errors.Wrap(err, fmt.Sprintf("error uploading %s to %s", task.sourceFilename, storage))
				}
			}
		} else {
			// Handle reader upload (for LinkFromPool)
			err = storage.putFile(task.path, task.sourceReader, task.sourceMD5)
			if err != nil {
				err = errors.Wrap(err, fmt.Sprintf("error uploading to %s", storage))
			}
		}

		if err != nil {
			// Send error to error channel (non-blocking)
			select {
			case storage.uploadErrors <- err:
			default:
			}
		}

		storage.uploadWg.Done()
	}
}

// NewPublishedStorage creates new instance of PublishedStorage with specified S3 access
// keys, region and bucket name
func NewPublishedStorage(
	accessKey, secretKey, sessionToken, region, endpoint, bucket, defaultACL, prefix, storageClass, encryptionMethod string,
	plusWorkaround, disableMultiDel, _, forceVirtualHostedStyle, debug bool, concurrentUploads int, uploadQueueSize int) (*PublishedStorage, error) {

	opts := []func(*config.LoadOptions) error{config.WithRegion(region)}
	if accessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)))
	}

	if debug {
		opts = append(opts, config.WithLogger(&logger{}))
	}

	config, err := config.LoadDefaultConfig(context.TODO(), opts...)
	if err != nil {
		return nil, err
	}

	result, err := NewPublishedStorageRaw(bucket, defaultACL, prefix, storageClass,
		encryptionMethod, plusWorkaround, disableMultiDel, forceVirtualHostedStyle, &config, endpoint, concurrentUploads, uploadQueueSize)

	return result, err
}

// String returns the storage as string
func (storage *PublishedStorage) String() string {
	return fmt.Sprintf("S3: %s:%s/%s", storage.config.Region, storage.bucket, storage.prefix)
}

// MkDir creates directory recursively under public path
func (storage *PublishedStorage) MkDir(_ string) error {
	// no op for S3
	return nil
}

// PutFile puts file into published storage at specified path
func (storage *PublishedStorage) PutFile(path string, sourceFilename string) error {
	// If concurrent uploads are disabled, use the original implementation
	if storage.concurrentUploads == 0 {
		var (
			source *os.File
			err    error
		)
		source, err = os.Open(sourceFilename)
		if err != nil {
			return err
		}
		defer func() { _ = source.Close() }()

		log.Debug().Msgf("S3: PutFile '%s'", path)
		err = storage.putFile(path, source, "")
		if err != nil {
			err = errors.Wrap(err, fmt.Sprintf("error uploading %s to %s", sourceFilename, storage))
		}

		return err
	}

	// Concurrent upload path
	log.Debug().Msgf("S3: PutFile '%s' (concurrent)", path)

	// Check for any previous errors
	select {
	case err := <-storage.uploadErrors:
		return err
	default:
	}

	// Queue the upload task
	task := &uploadTask{
		path:           path,
		sourceFilename: sourceFilename,
		isFile:         true,
	}

	storage.uploadWg.Add(1)
	select {
	case storage.uploadQueue <- task:
		// Task queued successfully
		return nil
	case err := <-storage.uploadErrors:
		storage.uploadWg.Done()
		return err
	}
}

// getMD5 retrieves MD5 stored in the metadata, if any
func (storage *PublishedStorage) getMD5(path string) (string, error) {
	params := &s3.HeadObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(filepath.Join(storage.prefix, path)),
	}
	output, err := storage.s3.HeadObject(context.TODO(), params)
	if err != nil {
		return "", err
	}

	return output.Metadata["Md5"], nil
}

// putFile uploads file-like object to
func (storage *PublishedStorage) putFile(path string, source io.ReadSeeker, sourceMD5 string) error {
	params := &s3.PutObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(filepath.Join(storage.prefix, path)),
		Body:   source,
		ACL:    storage.acl,
	}
	if storage.storageClass != "" {
		params.StorageClass = types.StorageClass(storage.storageClass)
	}
	if storage.encryptionMethod != "" {
		params.ServerSideEncryption = types.ServerSideEncryption(storage.encryptionMethod)
	}
	if sourceMD5 != "" {
		params.Metadata = map[string]string{
			"Md5": sourceMD5,
		}
	}

	_, err := storage.s3.PutObject(context.TODO(), params)
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

	log.Debug().Msgf("S3: Remove '%s'", path)
	if _, err := storage.s3.DeleteObject(context.TODO(), params); err != nil {
		var notFoundErr *smithy.GenericAPIError
		if errors.As(err, &notFoundErr) && notFoundErr.Code == "NoSuchBucket" {
			// ignore 'no such bucket' errors on removal
			return nil
		}
		return errors.Wrap(err, fmt.Sprintf("error deleting %s from %s", path, storage))
	}

	if storage.plusWorkaround && strings.Contains(path, "+") {
		// try to remove workaround version, but don't care about result
		_ = storage.Remove(strings.Replace(path, "+", " ", -1))
	}

	// Thread-safe cache delete
	storage.pathCacheMutex.Lock()
	delete(storage.pathCache, path)
	storage.pathCacheMutex.Unlock()

	return nil
}

// RemoveDirs removes directory structure under public path
func (storage *PublishedStorage) RemoveDirs(path string, _ aptly.Progress) error {
	const page = 1000

	filelist, _, err := storage.internalFilelist(path, false)
	if err != nil {
		if errors.Is(err, &types.NoSuchBucket{}) {
			// ignore 'no such bucket' errors on removal
			return nil
		}
		return err
	}

	log.Debug().Msgf("S3: RemoveDirs '%s'", path)
	if storage.disableMultiDel {
		for i := range filelist {
			params := &s3.DeleteObjectInput{
				Bucket: aws.String(storage.bucket),
				Key:    aws.String(filepath.Join(storage.prefix, path, filelist[i])),
			}
			_, err := storage.s3.DeleteObject(context.TODO(), params)
			if err != nil {
				return fmt.Errorf("error deleting path %s from %s: %s", filelist[i], storage, err)
			}
			// Thread-safe cache delete
			storage.pathCacheMutex.Lock()
			delete(storage.pathCache, filepath.Join(path, filelist[i]))
			storage.pathCacheMutex.Unlock()
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

			paths := make([]types.ObjectIdentifier, len(part))
			for i := range part {
				paths[i] = types.ObjectIdentifier{
					Key: aws.String(filepath.Join(storage.prefix, path, part[i])),
				}
			}

			quiet := true
			params := &s3.DeleteObjectsInput{
				Bucket: aws.String(storage.bucket),
				Delete: &types.Delete{
					Objects: paths,
					Quiet:   &quiet,
				},
			}

			_, err := storage.s3.DeleteObjects(context.TODO(), params)
			if err != nil {
				return fmt.Errorf("error deleting multiple paths from %s: %s", storage, err)
			}
			// Thread-safe cache delete for batch operations
			storage.pathCacheMutex.Lock()
			for i := range part {
				delete(storage.pathCache, filepath.Join(path, part[i]))
			}
			storage.pathCacheMutex.Unlock()
		}
	}

	return nil
}

// LinkFromPool links package file from pool to dist's pool location
//
// publishedPrefix is desired prefix for the location in the pool.
// publishedRelPath is desired location in pool (like pool/component/liba/libav/)
// sourcePool is instance of aptly.PackagePool
// sourcePath is filepath to package file in package pool
//
// LinkFromPool returns relative path for the published file to be included in package index
func (storage *PublishedStorage) LinkFromPool(publishedPrefix, publishedRelPath, fileName string, sourcePool aptly.PackagePool,
	sourcePath string, sourceChecksums utils.ChecksumInfo, force bool) error {

	publishedDirectory := filepath.Join(publishedPrefix, publishedRelPath)
	relPath := filepath.Join(publishedDirectory, fileName)
	poolPath := filepath.Join(storage.prefix, relPath)

	// Thread-safe cache initialization
	storage.pathCacheMutex.RLock()
	cacheExists := storage.pathCache != nil
	storage.pathCacheMutex.RUnlock()

	if !cacheExists {
		storage.pathCacheMutex.Lock()
		// Double-check pattern to avoid race condition
		if storage.pathCache == nil {
			paths, md5s, err := storage.internalFilelist(filepath.Join(publishedPrefix, "pool"), true)
			if err != nil {
				storage.pathCacheMutex.Unlock()
				return errors.Wrap(err, "error caching paths under prefix")
			}

			storage.pathCache = make(map[string]string, len(paths))

			for i := range paths {
				storage.pathCache[filepath.Join("pool", paths[i])] = md5s[i]
			}
		}
		storage.pathCacheMutex.Unlock()
	}

	// Thread-safe cache read
	storage.pathCacheMutex.RLock()
	destinationMD5, exists := storage.pathCache[relPath]
	storage.pathCacheMutex.RUnlock()
	sourceMD5 := sourceChecksums.MD5

	if exists {
		if sourceMD5 == "" {
			return fmt.Errorf("unable to compare object, MD5 checksum missing")
		}

		if len(destinationMD5) != 32 || storage.encryptByDefault {
			// doesn’t look like a valid MD5,
			// attempt to fetch one from the metadata
			var err error
			destinationMD5, err = storage.getMD5(relPath)
			if err != nil {
				err = errors.Wrap(err, fmt.Sprintf("error verifying MD5 for %s: %s", storage, poolPath))
				return err
			}
			// Thread-safe cache write
			storage.pathCacheMutex.Lock()
			storage.pathCache[relPath] = destinationMD5
			storage.pathCacheMutex.Unlock()
		}

		if destinationMD5 == sourceMD5 {
			return nil
		}

		if !force {
			return fmt.Errorf("error putting file to %s: file already exists and is different: %s", poolPath, storage)
		}
	}

	// If concurrent uploads are disabled, use the original implementation
	if storage.concurrentUploads == 0 {
		source, err := sourcePool.Open(sourcePath)
		if err != nil {
			return err
		}
		defer func() { _ = source.Close() }()

		log.Debug().Msgf("S3: LinkFromPool '%s'", relPath)
		err = storage.putFile(relPath, source, sourceMD5)
		if err == nil {
			// Thread-safe cache write
			storage.pathCacheMutex.Lock()
			storage.pathCache[relPath] = sourceMD5
			storage.pathCacheMutex.Unlock()
		} else {
			err = errors.Wrap(err, fmt.Sprintf("error uploading %s to %s: %s", sourcePath, storage, poolPath))
		}

		return err
	}

	// Concurrent upload path
	log.Debug().Msgf("S3: LinkFromPool '%s' (concurrent)", relPath)

	// Check for any previous errors
	select {
	case err := <-storage.uploadErrors:
		return err
	default:
	}

	// Open the source file to create a copy for the worker
	source, err := sourcePool.Open(sourcePath)
	if err != nil {
		return err
	}

	// Read the entire content into memory to avoid concurrent access issues
	content, err := io.ReadAll(source)
	_ = source.Close()
	if err != nil {
		return err
	}

	// Create a new reader from the content
	reader := bytes.NewReader(content)

	// Queue the upload task
	task := &uploadTask{
		path:         relPath,
		sourceReader: reader,
		sourceMD5:    sourceMD5,
		isFile:       false,
	}

	storage.uploadWg.Add(1)
	select {
	case storage.uploadQueue <- task:
		// Task queued successfully
		// Update cache optimistically
		storage.pathCacheMutex.Lock()
		storage.pathCache[relPath] = sourceMD5
		storage.pathCacheMutex.Unlock()
		return nil
	case err := <-storage.uploadErrors:
		storage.uploadWg.Done()
		return err
	}
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

	maxKeys := int32(1000)
	params := &s3.ListObjectsV2Input{
		Bucket:  aws.String(storage.bucket),
		Prefix:  aws.String(prefix),
		MaxKeys: &maxKeys,
	}

	p := s3.NewListObjectsV2Paginator(storage.s3, params)
	for i := 1; p.HasMorePages(); i++ {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get page %d: %w", i, err)
		}

		// Log the objects found
		for _, key := range page.Contents {
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
	}

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
		ACL:        storage.acl,
	}

	if storage.storageClass != "" {
		params.StorageClass = storage.storageClass
	}
	if storage.encryptionMethod != "" {
		params.ServerSideEncryption = storage.encryptionMethod
	}

	log.Debug().Msgf("S3: RenameFile %s -> %s", oldName, newName)
	_, err := storage.s3.CopyObject(context.TODO(), params)
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
		ACL:        types.ObjectCannedACL(storage.acl),
		Metadata: map[string]string{
			"SymLink": src,
		},
		MetadataDirective: types.MetadataDirective("REPLACE"),
	}

	if storage.storageClass != "" {
		params.StorageClass = types.StorageClass(storage.storageClass)
	}
	if storage.encryptionMethod != "" {
		params.ServerSideEncryption = types.ServerSideEncryption(storage.encryptionMethod)
	}

	log.Debug().Msgf("S3: SymLink %s -> %s", src, dst)
	_, err := storage.s3.CopyObject(context.TODO(), params)
	if err != nil {
		return fmt.Errorf("error symlinking %s -> %s in %s: %s", src, dst, storage, err)
	}

	return err
}

// HardLink using symlink functionality as hard links do not exist
func (storage *PublishedStorage) HardLink(src string, dst string) error {
	log.Debug().Msgf("S3: HardLink %s -> %s", src, dst)
	return storage.SymLink(src, dst)
}

// Flush waits for all concurrent uploads to complete and returns any errors
func (storage *PublishedStorage) Flush() error {
	if storage.concurrentUploads == 0 {
		// Nothing to flush if concurrent uploads are disabled
		return nil
	}

	// Wait for all uploads to complete
	storage.uploadWg.Wait()

	// Check for any errors
	select {
	case err := <-storage.uploadErrors:
		return err
	default:
		return nil
	}
}

// FileExists returns true if path exists
func (storage *PublishedStorage) FileExists(path string) (bool, error) {
	params := &s3.HeadObjectInput{
		Bucket: aws.String(storage.bucket),
		Key:    aws.String(filepath.Join(storage.prefix, path)),
	}
	_, err := storage.s3.HeadObject(context.TODO(), params)
	if err != nil {
		var notFoundErr *types.NotFound
		if errors.As(err, &notFoundErr) {
			return false, nil
		}

		// falback in case the above condidition fails
		var opErr *smithy.OperationError
		if errors.As(err, &opErr) {
			var ae smithy.APIError
			if errors.As(err, &ae) {
				if ae.ErrorCode() == "NotFound" {
					return false, nil
				}
			}
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
	output, err := storage.s3.HeadObject(context.TODO(), params)
	if err != nil {
		return "", err
	}

	return output.Metadata["SymLink"], nil
}
