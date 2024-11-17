package azure

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
	"github.com/pkg/errors"
)

type PackagePool struct {
	az *azContext
}

// Check interface
var (
	_ aptly.PackagePool = (*PackagePool)(nil)
)

// NewPackagePool creates published storage from Azure storage credentials
func NewPackagePool(accountName, accountKey, container, prefix, endpoint string) (*PackagePool, error) {
	azctx, err := newAzContext(accountName, accountKey, container, prefix, endpoint)
	if err != nil {
		return nil, err
	}

	return &PackagePool{az: azctx}, nil
}

// String
func (pool *PackagePool) String() string {
	return pool.az.String()
}

func (pool *PackagePool) buildPoolPath(filename string, checksums *utils.ChecksumInfo) string {
	hash := checksums.SHA256
	// Use the same path as the file pool, for compat reasons.
	return filepath.Join(hash[0:2], hash[2:4], hash[4:32]+"_"+filename)
}

func (pool *PackagePool) ensureChecksums(
	poolPath string,
	checksumStorage aptly.ChecksumStorage,
) (*utils.ChecksumInfo, error) {
	targetChecksums, err := checksumStorage.Get(poolPath)
	if err != nil {
		return nil, err
	}

	if targetChecksums == nil {
		// we don't have checksums stored yet for this file
		blob := pool.az.blobURL(poolPath)
		download, err := blob.Download(context.Background(), 0, 0, azblob.BlobAccessConditions{}, false, azblob.ClientProvidedKeyOptions{})
		if err != nil {
			if isBlobNotFound(err) {
				return nil, nil
			}

			return nil, errors.Wrapf(err, "error downloading blob at %s", poolPath)
		}

		targetChecksums = &utils.ChecksumInfo{}
		*targetChecksums, err = utils.ChecksumsForReader(download.Body(azblob.RetryReaderOptions{}))
		if err != nil {
			return nil, errors.Wrapf(err, "error checksumming blob at %s", poolPath)
		}

		err = checksumStorage.Update(poolPath, targetChecksums)
		if err != nil {
			return nil, err
		}
	}

	return targetChecksums, nil
}

func (pool *PackagePool) FilepathList(progress aptly.Progress) ([]string, error) {
	if progress != nil {
		progress.InitBar(0, false, aptly.BarGeneralBuildFileList)
		defer progress.ShutdownBar()
	}

	paths, _, err := pool.az.internalFilelist("", progress)
	return paths, err
}

func (pool *PackagePool) LegacyPath(_ string, _ *utils.ChecksumInfo) (string, error) {
	return "", errors.New("Azure package pool does not support legacy paths")
}

func (pool *PackagePool) Size(path string) (int64, error) {
	blob := pool.az.blobURL(path)
	props, err := blob.GetProperties(context.Background(), azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return 0, errors.Wrapf(err, "error examining %s from %s", path, pool)
	}

	return props.ContentLength(), nil
}

func (pool *PackagePool) Open(path string) (aptly.ReadSeekerCloser, error) {
	blob := pool.az.blobURL(path)

	temp, err := os.CreateTemp("", "blob-download")
	if err != nil {
		return nil, errors.Wrap(err, "error creating temporary file for blob download")
	}

	defer os.Remove(temp.Name())

	err = azblob.DownloadBlobToFile(context.Background(), blob, 0, 0, temp, azblob.DownloadFromBlobOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "error downloading blob at %s", path)
	}

	return temp, nil
}

func (pool *PackagePool) Remove(path string) (int64, error) {
	blob := pool.az.blobURL(path)
	props, err := blob.GetProperties(context.Background(), azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return 0, errors.Wrapf(err, "error getting props of %s from %s", path, pool)
	}

	_, err = blob.Delete(context.Background(), azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	if err != nil {
		return 0, errors.Wrapf(err, "error deleting %s from %s", path, pool)
	}

	return props.ContentLength(), nil
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
	blob := pool.az.blobURL(path)
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
	defer source.Close()

	err = pool.az.putFile(blob, source, checksums.MD5)
	if err != nil {
		return "", err
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

	size, err := pool.Size(poolPath)
	if err != nil {
		return "", false, err
	} else if size != checksums.Size {
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
