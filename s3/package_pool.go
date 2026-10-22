package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go"
	"github.com/pkg/errors"
)

type PackagePool struct {
	s3               *s3.Client
	bucket           string
	prefix           string
	acl              types.ObjectCannedACL
	storageClass     types.StorageClass
	encryptionMethod types.ServerSideEncryption
}

// Check interface
var (
	_ aptly.PackagePool = (*PackagePool)(nil)
)

func NewPackagePool(accessKey, secretKey, sessionToken, bucket, prefix, defaultACL,
	storageClass, encryptionMethod, region, endpoint string, forceVirtualHostedStyle, debug bool) (*PackagePool, error) {

	opts := []func(*config.LoadOptions) error{config.WithRegion(region)}
	if accessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)))
	}

	if debug {
		opts = append(opts, config.WithLogger(&logger{}))
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load AWS configuration")
	}

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

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = !forceVirtualHostedStyle
		o.HTTPSignerV4 = signer.NewSigner()
		o.BaseEndpoint = baseEndpoint
	})

	return &PackagePool{
		s3:               client,
		bucket:           bucket,
		prefix:           prefix,
		acl:              acl,
		storageClass:     types.StorageClass(storageClass),
		encryptionMethod: types.ServerSideEncryption(encryptionMethod),
	}, nil
}

// String returns the storage as string
func (pool *PackagePool) String() string {
	return fmt.Sprintf("S3: %s/%s", pool.bucket, pool.prefix)
}

func (pool *PackagePool) buildPoolPath(filename string, checksums *utils.ChecksumInfo) string {
	hash := checksums.SHA256
	// Use the same path as the file pool, for compat reasons.
	return filepath.Join(hash[0:2], hash[2:4], hash[4:32]+"_"+filename)
}

func (pool *PackagePool) ensureChecksums(poolPath string, checksumStorage aptly.ChecksumStorage) (*utils.ChecksumInfo, error) {
	targetChecksums, err := checksumStorage.Get(poolPath)
	if err != nil {
		return nil, err
	}

	if targetChecksums == nil {
		key := pool.poolKey(poolPath)

		// we don't have checksums stored yet for this file
		output, err := pool.s3.GetObject(context.Background(), &s3.GetObjectInput{
			Key:    &key,
			Bucket: &pool.bucket,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, errors.Wrapf(err, "error downloading object at %s from %s", poolPath, pool)
		}
		defer func() { _ = output.Body.Close() }()

		targetChecksums = &utils.ChecksumInfo{}
		*targetChecksums, err = utils.ChecksumsForReader(output.Body)
		if err != nil {
			return nil, errors.Wrapf(err, "error checksumming blob at %s from %s", poolPath, pool)
		}

		err = checksumStorage.Update(poolPath, targetChecksums)
		if err != nil {
			return nil, errors.Wrap(err, "error updating checksumStorage")
		}
	}

	return targetChecksums, nil
}

// FilepathList returns file paths of all the files in the pool
func (pool *PackagePool) FilepathList(progress aptly.Progress) ([]string, error) {
	if progress != nil {
		progress.InitBar(0, false, aptly.BarGeneralBuildFileList)
		defer progress.ShutdownBar()
	}

	paths := make([]string, 0, 1024)
	prefix := pool.prefix
	if prefix != "" {
		prefix += "/"
	}

	maxKeys := int32(1000)
	params := &s3.ListObjectsV2Input{
		Bucket:  &pool.bucket,
		Prefix:  &prefix,
		MaxKeys: &maxKeys,
	}

	p := s3.NewListObjectsV2Paginator(pool.s3, params)
	for i := 1; p.HasMorePages(); i++ {
		page, err := p.NextPage(context.TODO())
		if err != nil {
			return nil, errors.Wrapf(err, "error listing pool %s (page %d)", pool, i)
		}

		for _, key := range page.Contents {
			if prefix == "" {
				paths = append(paths, *key.Key)
			} else {
				paths = append(paths, (*key.Key)[len(prefix):])
			}
		}
	}

	return paths, nil
}

func (pool *PackagePool) LegacyPath(_ string, _ *utils.ChecksumInfo) (string, error) {
	return "", errors.New("S3 package pool does not support legacy paths")
}

func (pool *PackagePool) Size(path string) (int64, error) {
	headObject, found, err := pool.headObject(path)
	if err != nil {
		return 0, errors.Wrapf(err, "error getting headObject for %s", path)
	}
	if !found {
		return 0, fmt.Errorf("%s not found", path)
	}

	return *headObject.ContentLength, nil
}

func (pool *PackagePool) Open(path string) (aptly.ReadSeekerCloser, error) {
	key := pool.poolKey(path)
	output, err := pool.s3.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &pool.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "error downloading object %s from %s", path, pool)
	}
	defer func() { _ = output.Body.Close() }()

	temp, err := os.CreateTemp("", "s3-pool-download")
	if err != nil {
		return nil, errors.Wrapf(err, "error creating tempfile for %s", path)
	}
	defer func() { _ = os.Remove(temp.Name()) }()

	_, err = io.Copy(temp, output.Body)
	if err != nil {
		_ = temp.Close()
		return nil, errors.Wrapf(err, "error downloading object %s from %s", path, pool)
	}

	_, err = temp.Seek(0, io.SeekStart)
	if err != nil {
		_ = temp.Close()
		return nil, errors.Wrapf(err, "error seeking tempfile for %s", path)
	}

	return temp, nil
}

func (pool *PackagePool) Remove(path string) (int64, error) {
	headObject, found, err := pool.headObject(path)
	if err != nil {
		return 0, errors.Wrapf(err, "error getting headObject for %s", path)
	}
	if !found {
		return 0, fmt.Errorf("%s not found in %s", path, pool)
	}
	key := pool.poolKey(path)
	_, err = pool.s3.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: &pool.bucket,
		Key:    &key,
	})
	if err != nil {
		return 0, errors.Wrapf(err, "error deleting %s from %s", path, pool)
	}

	return *headObject.ContentLength, nil
}

func (pool *PackagePool) Import(srcPath, basename string, checksums *utils.ChecksumInfo, _ bool, checksumStorage aptly.ChecksumStorage) (string, error) {
	if checksums.MD5 == "" || checksums.SHA256 == "" || checksums.SHA512 == "" {
		// need to update checksums, MD5 and SHA256 should be always defined
		var err error
		*checksums, err = utils.ChecksumsForFile(srcPath)
		if err != nil {
			return "", err
		}
	}

	path := pool.buildPoolPath(basename, checksums)
	targetChecksums, err := pool.ensureChecksums(path, checksumStorage)
	if err != nil {
		return "", err
	} else if targetChecksums != nil {
		// target already exists
		*checksums = *targetChecksums
		return path, nil
	}

	source, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = source.Close() }()

	err = pool.putFile(path, source)
	if err != nil {
		return "", errors.Wrapf(err, "error uploading %s to %s", srcPath, pool)
	}

	if !checksums.Complete() {
		// need full checksums here
		*checksums, err = utils.ChecksumsForFile(srcPath)
		if err != nil {
			return "", err
		}
	}

	err = checksumStorage.Update(path, checksums)
	if err != nil {
		return "", err
	}

	return path, nil
}

func (pool *PackagePool) Verify(poolPath, basename string, checksums *utils.ChecksumInfo, checksumStorage aptly.ChecksumStorage) (string, bool, error) {
	if poolPath == "" {
		if checksums.SHA256 != "" {
			poolPath = pool.buildPoolPath(basename, checksums)
		} else {
			// No checksums or pool path, so no idea what file to look for.
			return "", false, nil
		}
	}

	output, found, err := pool.headObject(poolPath)
	if err != nil {
		return "", false, errors.Wrapf(err, "error examining %s from %s", poolPath, pool)
	}
	if !found || *output.ContentLength != checksums.Size {
		return "", false, nil
	}

	targetChecksums, err := pool.ensureChecksums(poolPath, checksumStorage)
	if err != nil {
		return "", false, err
	} else if targetChecksums == nil {
		return "", false, nil
	}

	if checksums.MD5 != "" && targetChecksums.MD5 != checksums.MD5 ||
		checksums.SHA256 != "" && targetChecksums.SHA256 != checksums.SHA256 {
		// wrong file?
		return "", false, nil
	}

	// fill back checksums
	*checksums = *targetChecksums
	return poolPath, true, nil
}

func (pool *PackagePool) putFile(path string, source io.ReadSeeker) error {
	key := pool.poolKey(path)
	params := &s3.PutObjectInput{
		Bucket: &pool.bucket,
		Key:    &key,
		Body:   source,
		ACL:    pool.acl,
	}

	if pool.storageClass != "" {
		params.StorageClass = pool.storageClass
	}
	if pool.encryptionMethod != "" {
		params.ServerSideEncryption = pool.encryptionMethod
	}

	_, err := pool.s3.PutObject(context.Background(), params)
	return err
}

// headObject returns object metadata, with found=false (and no error) when the
// object does not exist
func (pool *PackagePool) headObject(path string) (*s3.HeadObjectOutput, bool, error) {
	key := pool.poolKey(path)
	output, err := pool.s3.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: &pool.bucket,
		Key:    &key,
	})

	if err != nil {
		if isNotFound(err) {
			return nil, false, nil
		}
		return nil, false, err
	}

	return output, true, nil
}

func (pool *PackagePool) poolKey(path string) string {
	return filepath.Join(pool.prefix, path)
}

func isNotFound(err error) bool {
	var ae smithy.APIError
	if errors.As(err, &ae) {
		switch ae.ErrorCode() {
		case "NoSuchKey", "NotFound":
			return true
		case "NoSuchBucket", "AccessDenied":
			// also 404/403 — but these are real failures, not a missing object
			return false
		}
	}

	var re *awshttp.ResponseError
	if errors.As(err, &re) {
		return re.HTTPStatusCode() == http.StatusNotFound
	}

	return false
}
