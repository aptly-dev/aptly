package azure

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/lease"
	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// PublishedStorage abstract file system with published files (actually hosted on Azure)
type PublishedStorage struct {
	// FIXME: unused ???? prefix    string
	az        *azContext
	pathCache map[string]map[string]string
}

// Check interface
var (
	_ aptly.PublishedStorage = (*PublishedStorage)(nil)
)

// NewPublishedStorage creates published storage from Azure storage credentials
func NewPublishedStorage(accountName, accountKey, container, prefix, endpoint string) (*PublishedStorage, error) {
	azctx, err := newAzContext(accountName, accountKey, container, prefix, endpoint)
	if err != nil {
		return nil, err
	}

	return &PublishedStorage{az: azctx}, nil
}

// String returns the storage as string
func (storage *PublishedStorage) String() string {
	return storage.az.String()
}

// MkDir creates directory recursively under public path
func (storage *PublishedStorage) MkDir(_ string) error {
	// no op for Azure
	return nil
}

// PutFile puts file into published storage at specified path
func (storage *PublishedStorage) PutFile(path string, sourceFilename string) error {
	var (
		source *os.File
		err    error
	)

	sourceMD5, err := utils.MD5ChecksumForFile(sourceFilename)
	if err != nil {
		return err
	}

	source, err = os.Open(sourceFilename)
	if err != nil {
		return err
	}
	defer func() { _ = source.Close() }()

	err = storage.az.putFile(path, source, sourceMD5)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("error uploading %s to %s", sourceFilename, storage))
	}

	return err
}

// RemoveDirs removes directory structure under public path
func (storage *PublishedStorage) RemoveDirs(path string, _ aptly.Progress) error {
	path = storage.az.blobPath(path)
	filelist, err := storage.Filelist(path)
	if err != nil {
		return err
	}

	for _, filename := range filelist {
		blob := filepath.Join(path, filename)
		_, err := storage.az.client.DeleteBlob(context.Background(), storage.az.container, blob, nil)
		if err != nil {
			return fmt.Errorf("error deleting path %s from %s: %s", filename, storage, err)
		}
	}

	return nil
}

// Remove removes single file under public path
func (storage *PublishedStorage) Remove(path string) error {
	path = storage.az.blobPath(path)
	_, err := storage.az.client.DeleteBlob(context.Background(), storage.az.container, path, nil)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("error deleting %s from %s: %s", path, storage, err))
	}
	return err
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

	relFilePath := filepath.Join(publishedRelPath, fileName)
	prefixRelFilePath := filepath.Join(publishedPrefix, relFilePath)
	poolPath := storage.az.blobPath(prefixRelFilePath)

	if storage.pathCache == nil {
		storage.pathCache = make(map[string]map[string]string)
	}
	pathCache := storage.pathCache[publishedPrefix]
	if pathCache == nil {
		paths, md5s, err := storage.az.internalFilelist(publishedPrefix, nil)
		if err != nil {
			return fmt.Errorf("error caching paths under prefix: %s", err)
		}

		pathCache = make(map[string]string, len(paths))

		for i := range paths {
			pathCache[paths[i]] = md5s[i]
		}
		storage.pathCache[publishedPrefix] = pathCache
	}

	destinationMD5, exists := pathCache[relFilePath]
	sourceMD5 := sourceChecksums.MD5

	if exists {
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
	defer func() { _ = source.Close() }()

	err = storage.az.putFile(relFilePath, source, sourceMD5)
	if err == nil {
		pathCache[relFilePath] = sourceMD5
	} else {
		err = errors.Wrap(err, fmt.Sprintf("error uploading %s to %s: %s", sourcePath, storage, poolPath))
	}

	return err
}

// Filelist returns list of files under prefix
func (storage *PublishedStorage) Filelist(prefix string) ([]string, error) {
	paths, _, err := storage.az.internalFilelist(prefix, nil)
	return paths, err
}

// Internal copy or move implementation
func (storage *PublishedStorage) internalCopyOrMoveBlob(src, dst string, metadata map[string]*string, move bool) error {
	const leaseDuration = 30
	leaseID := uuid.NewString()

	serviceClient := storage.az.client.ServiceClient()
	containerClient := serviceClient.NewContainerClient(storage.az.container)
	srcBlobClient := containerClient.NewBlobClient(src)
	blobLeaseClient, err := lease.NewBlobClient(srcBlobClient, &lease.BlobClientOptions{LeaseID: to.Ptr(leaseID)})
	if err != nil {
		return fmt.Errorf("error acquiring lease on source blob %s", src)
	}

	_, err = blobLeaseClient.AcquireLease(context.Background(), leaseDuration, nil)
	if err != nil {
		return fmt.Errorf("error acquiring lease on source blob %s", src)
	}
	defer func() {
		_, _ = blobLeaseClient.BreakLease(context.Background(), &lease.BlobBreakOptions{BreakPeriod: to.Ptr(int32(60))})
	}()

	dstBlobClient := containerClient.NewBlobClient(dst)
	copyResp, err := dstBlobClient.StartCopyFromURL(context.Background(), srcBlobClient.URL(), &blob.StartCopyFromURLOptions{
		Metadata: metadata,
	})

	if err != nil {
		return fmt.Errorf("error copying %s -> %s in %s: %s", src, dst, storage, err)
	}

	copyStatus := *copyResp.CopyStatus
	for {
		if copyStatus == blob.CopyStatusTypeSuccess {
			if move {
				_, err := storage.az.client.DeleteBlob(context.Background(), storage.az.container, src, &blob.DeleteOptions{
					AccessConditions: &blob.AccessConditions{
						LeaseAccessConditions: &blob.LeaseAccessConditions{
							LeaseID: &leaseID,
						},
					},
				})
				return err
			}
			return nil
		} else if copyStatus == blob.CopyStatusTypePending {
			time.Sleep(1 * time.Second)
			getMetadata, err := dstBlobClient.GetProperties(context.TODO(), nil)
			if err != nil {
				return fmt.Errorf("error getting copy progress %s", dst)
			}
			copyStatus = *getMetadata.CopyStatus

			_, err = blobLeaseClient.RenewLease(context.Background(), nil)
			if err != nil {
				return fmt.Errorf("error renewing source blob lease %s", src)
			}
		} else {
			return fmt.Errorf("error copying %s -> %s in %s: %s", dst, src, storage, copyStatus)
		}
	}
}

// RenameFile renames (moves) file
func (storage *PublishedStorage) RenameFile(oldName, newName string) error {
	return storage.internalCopyOrMoveBlob(oldName, newName, nil, true /* move */)
}

// SymLink creates a copy of src file and adds link information as meta data
func (storage *PublishedStorage) SymLink(src string, dst string) error {
	metadata := make(map[string]*string)
	metadata["SymLink"] = &src
	return storage.internalCopyOrMoveBlob(src, dst, metadata, false /* do not remove src */)
}

// HardLink using symlink functionality as hard links do not exist
func (storage *PublishedStorage) HardLink(src string, dst string) error {
	return storage.SymLink(src, dst)
}

// FileExists returns true if path exists
func (storage *PublishedStorage) FileExists(path string) (bool, error) {
	serviceClient := storage.az.client.ServiceClient()
	containerClient := serviceClient.NewContainerClient(storage.az.container)
	blobClient := containerClient.NewBlobClient(path)
	_, err := blobClient.GetProperties(context.Background(), nil)
	if err != nil {
		if isBlobNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("error checking if blob %s exists: %v", path, err)
	}
	return true, nil
}

// ReadLink returns the symbolic link pointed to by path.
// This simply reads text file created with SymLink
func (storage *PublishedStorage) ReadLink(path string) (string, error) {
	serviceClient := storage.az.client.ServiceClient()
	containerClient := serviceClient.NewContainerClient(storage.az.container)
	blobClient := containerClient.NewBlobClient(path)
	props, err := blobClient.GetProperties(context.Background(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to get blob properties: %v", err)
	}

	metadata := props.Metadata
	if originalBlob, exists := metadata["original_blob"]; exists {
		return *originalBlob, nil
	}
	return "", fmt.Errorf("error reading link %s: %v", path, err)
}
