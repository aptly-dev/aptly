package azure

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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
	pathCache map[string]string
}

// Check interface
var (
	_ aptly.PublishedStorage = (*PublishedStorage)(nil)
)

func isEmulatorEndpoint(endpoint string) bool {
	if h, _, err := net.SplitHostPort(endpoint); err == nil {
		endpoint = h
	}
	if endpoint == "localhost" {
		return true
	}
	// For IPv6, there could be case where SplitHostPort fails for cannot finding port.
	// In this case, eliminate the '[' and ']' in the URL.
	// For details about IPv6 URL, please refer to https://tools.ietf.org/html/rfc2732
	if endpoint[0] == '[' && endpoint[len(endpoint)-1] == ']' {
		endpoint = endpoint[1 : len(endpoint)-1]
	}
	return net.ParseIP(endpoint) != nil
}

// NewPublishedStorage creates published storage from Azure storage credentials
func NewPublishedStorage(accountName, accountKey, container, prefix, endpoint string) (*PublishedStorage, error) {
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	if endpoint == "" {
		endpoint = "blob.core.windows.net"
	}

	var url *url.URL
	if isEmulatorEndpoint(endpoint) {
		url, err = url.Parse(fmt.Sprintf("http://%s/%s/%s", endpoint, accountName, container))
	} else {
		url, err = url.Parse(fmt.Sprintf("https://%s.%s/%s", accountName, endpoint, container))
	}
	if err != nil {
		return nil, err
	}

	containerURL := azblob.NewContainerURL(*url, azblob.NewPipeline(credential, azblob.PipelineOptions{}))

	result := &PublishedStorage{
		container: containerURL,
		prefix:    prefix,
	}

	return result, nil
}

// String
func (storage *PublishedStorage) String() string {
	return fmt.Sprintf("Azure: %s/%s", storage.container, storage.prefix)
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

	err = storage.putFile(path, source, sourceMD5)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("error uploading %s to %s", sourceFilename, storage))
	}

	return err
}

// putFile uploads file-like object to
func (storage *PublishedStorage) putFile(path string, source io.Reader, sourceMD5 string) error {
	path = filepath.Join(storage.prefix, path)

	blob := storage.container.NewBlockBlobURL(path)

	uploadOptions := azblob.UploadStreamToBlockBlobOptions{
		BufferSize: 4 * 1024 * 1024,
		MaxBuffers: 8,
	}
	if len(sourceMD5) > 0 {
		decodedMD5, err := hex.DecodeString(sourceMD5)
		if err != nil {
			return err
		}
		uploadOptions.BlobHTTPHeaders = azblob.BlobHTTPHeaders{
			ContentMD5: decodedMD5,
		}
	}

	_, err := azblob.UploadStreamToBlockBlob(
		context.Background(),
		source,
		blob,
		uploadOptions,
	)

	return err
}

// RemoveDirs removes directory structure under public path
func (storage *PublishedStorage) RemoveDirs(path string, _ aptly.Progress) error {
	filelist, err := storage.Filelist(path)
	if err != nil {
		return err
	}

	for _, filename := range filelist {
		blob := storage.container.NewBlobURL(filepath.Join(storage.prefix, path, filename))
		_, err := blob.Delete(context.Background(), azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
		if err != nil {
			return fmt.Errorf("error deleting path %s from %s: %s", filename, storage, err)
		}
	}

	return nil
}

// Remove removes single file under public path
func (storage *PublishedStorage) Remove(path string) error {
	blob := storage.container.NewBlobURL(filepath.Join(storage.prefix, path))
	_, err := blob.Delete(context.Background(), azblob.DeleteSnapshotsOptionNone, azblob.BlobAccessConditions{})
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("error deleting %s from %s: %s", path, storage, err))
	}
	return err
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
		paths, md5s, err := storage.internalFilelist("")
		if err != nil {
			return fmt.Errorf("error caching paths under prefix: %s", err)
		}

		storage.pathCache = make(map[string]string, len(paths))

		for i := range paths {
			storage.pathCache[paths[i]] = md5s[i]
		}
	}

	destinationMD5, exists := storage.pathCache[relPath]
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

	err = storage.putFile(relPath, source, sourceMD5)
	if err == nil {
		storage.pathCache[relPath] = sourceMD5
	} else {
		err = errors.Wrap(err, fmt.Sprintf("error uploading %s to %s: %s", sourcePath, storage, poolPath))
	}

	return err
}

func (storage *PublishedStorage) internalFilelist(prefix string) (paths []string, md5s []string, err error) {
	const delimiter = "/"
	paths = make([]string, 0, 1024)
	md5s = make([]string, 0, 1024)
	prefix = filepath.Join(storage.prefix, prefix)
	if prefix != "" {
		prefix += delimiter
	}

	for marker := (azblob.Marker{}); marker.NotDone(); {
		listBlob, err := storage.container.ListBlobsFlatSegment(
			context.Background(), marker, azblob.ListBlobsSegmentOptions{
				Prefix:     prefix,
				MaxResults: 1000,
				Details:    azblob.BlobListingDetails{Metadata: true}})
		if err != nil {
			return nil, nil, fmt.Errorf("error listing under prefix %s in %s: %s", prefix, storage, err)
		}

		marker = listBlob.NextMarker

		for _, blob := range listBlob.Segment.BlobItems {
			if prefix == "" {
				paths = append(paths, blob.Name)
			} else {
				paths = append(paths, blob.Name[len(prefix):])
			}
			md5s = append(md5s, fmt.Sprintf("%x", blob.Properties.ContentMD5))
		}
	}

	return paths, md5s, nil
}

// Filelist returns list of files under prefix
func (storage *PublishedStorage) Filelist(prefix string) ([]string, error) {
	paths, _, err := storage.internalFilelist(prefix)
	return paths, err
}

// Internal copy or move implementation
func (storage *PublishedStorage) internalCopyOrMoveBlob(src, dst string, metadata azblob.Metadata, move bool) error {
	const leaseDuration = 30

	dstBlobURL := storage.container.NewBlobURL(filepath.Join(storage.prefix, dst))
	srcBlobURL := storage.container.NewBlobURL(filepath.Join(storage.prefix, src))
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
	blob := storage.container.NewBlobURL(filepath.Join(storage.prefix, path))
	resp, err := blob.GetProperties(context.Background(), azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		storageError, ok := err.(azblob.StorageError)
		if ok && string(storageError.ServiceCode()) == string(azblob.StorageErrorCodeBlobNotFound) {
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
	blob := storage.container.NewBlobURL(filepath.Join(storage.prefix, path))
	resp, err := blob.GetProperties(context.Background(), azblob.BlobAccessConditions{}, azblob.ClientProvidedKeyOptions{})
	if err != nil {
		return "", err
	} else if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("error checking if blob %s exists %d", blob, resp.StatusCode())
	}
	return resp.NewMetadata()["SymLink"], nil
}
