package azure

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
	"github.com/pkg/errors"
)

// PublishedStorage abstract file system with published files (actually hosted on Azure)
type PublishedStorage struct {
	container azblob.ContainerURL
	prefix    string
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

// String
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
	defer source.Close()

	err = storage.az.putFile(storage.az.blobURL(path), source, sourceMD5)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("error uploading %s to %s", sourceFilename, storage))
	}

	return err
}

// RemoveDirs removes directory structure under public path
func (storage *PublishedStorage) RemoveDirs(path string, _ aptly.Progress) error {
	filelist, err := storage.Filelist(path)
	if err != nil {
		return err
	}

	for _, filename := range filelist {
		blob := storage.az.blobURL(filepath.Join(path, filename))
		_, err := blob.Delete(context.Background(), azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
		if err != nil {
			return fmt.Errorf("error deleting path %s from %s: %s", filename, storage, err)
		}
	}

	return nil
}

// Remove removes single file under public path
func (storage *PublishedStorage) Remove(path string) error {
	blob := storage.az.blobURL(path)
	_, err := blob.Delete(context.Background(), azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
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
	// prefixRelFilePath := filepath.Join(publishedPrefix, relFilePath)
	// FIXME: check how to integrate publishedPrefix:
	poolPath := storage.az.blobPath(fileName)

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
	defer source.Close()

	err = storage.az.putFile(storage.az.blobURL(relFilePath), source, sourceMD5)
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
func (storage *PublishedStorage) internalCopyOrMoveBlob(src, dst string, metadata azblob.Metadata, move bool) error {
	const leaseDuration = 30

	dstBlobURL := storage.az.blobURL(dst)
	srcBlobURL := storage.az.blobURL(src)
	leaseResp, err := srcBlobURL.AcquireLease(context.Background(), "", leaseDuration, azblob.ModifiedAccessConditions{})
	if err != nil || leaseResp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("error acquiring lease on source blob %s", srcBlobURL)
	}
	defer srcBlobURL.BreakLease(context.Background(), azblob.LeaseBreakNaturally, azblob.ModifiedAccessConditions{})
	srcBlobLeaseID := leaseResp.LeaseID()

	copyResp, err := dstBlobURL.StartCopyFromURL(
		context.Background(),
		srcBlobURL.URL(),
		metadata,
		azblob.ModifiedAccessConditions{},
		azblob.BlobAccessConditions{},
		azblob.DefaultAccessTier,
		nil)
	if err != nil {
		return fmt.Errorf("error copying %s -> %s in %s: %s", src, dst, storage, err)
	}

	copyStatus := copyResp.CopyStatus()
	for {
		if copyStatus == azblob.CopyStatusSuccess {
			if move {
				_, err = srcBlobURL.Delete(
					context.Background(),
					azblob.DeleteSnapshotsOptionNone,
					azblob.BlobAccessConditions{
						LeaseAccessConditions: azblob.LeaseAccessConditions{LeaseID: srcBlobLeaseID},
					})
				return err
			}
			return nil
		} else if copyStatus == azblob.CopyStatusPending {
			time.Sleep(1 * time.Second)
			blobPropsResp, err := dstBlobURL.GetProperties(
				context.Background(),
				azblob.BlobAccessConditions{LeaseAccessConditions: azblob.LeaseAccessConditions{LeaseID: srcBlobLeaseID}},
				azblob.ClientProvidedKeyOptions{})
			if err != nil {
				return fmt.Errorf("error getting destination blob properties %s", dstBlobURL)
			}
			copyStatus = blobPropsResp.CopyStatus()

			_, err = srcBlobURL.RenewLease(context.Background(), srcBlobLeaseID, azblob.ModifiedAccessConditions{})
			if err != nil {
				return fmt.Errorf("error renewing source blob lease %s", srcBlobURL)
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
	return storage.internalCopyOrMoveBlob(src, dst, azblob.Metadata{"SymLink": src}, false /* move */)
}

// HardLink using symlink functionality as hard links do not exist
func (storage *PublishedStorage) HardLink(src string, dst string) error {
	return storage.SymLink(src, dst)
}

// FileExists returns true if path exists
func (storage *PublishedStorage) FileExists(path string) (bool, error) {
	blob := storage.az.blobURL(path)
	resp, err := blob.GetProperties(context.Background(), azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		if isBlobNotFound(err) {
			return false, nil
		}
		return false, err
	} else if resp.StatusCode() == http.StatusOK {
		return true, nil
	}
	return false, fmt.Errorf("error checking if blob %s exists %d", blob, resp.StatusCode())
}

// ReadLink returns the symbolic link pointed to by path.
// This simply reads text file created with SymLink
func (storage *PublishedStorage) ReadLink(path string) (string, error) {
	blob := storage.az.blobURL(path)
	resp, err := blob.GetProperties(context.Background(), azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return "", err
	} else if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("error checking if blob %s exists %d", blob, resp.StatusCode())
	}
	return resp.NewMetadata()["SymLink"], nil
}
