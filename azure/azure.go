package azure

// Package azure handles publishing to Azure Storage

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/aptly-dev/aptly/aptly"
)

func isBlobNotFound(err error) bool {
	storageError, ok := err.(azblob.StorageError)
	return ok && storageError.ServiceCode() == azblob.ServiceCodeBlobNotFound
}

type azContext struct {
	container azblob.ContainerURL
	prefix    string
}

func newAzContext(accountName, accountKey, container, prefix, endpoint string) (*azContext, error) {
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.blob.core.windows.net", accountName)
	}

	url, err := url.Parse(fmt.Sprintf("%s/%s", endpoint, container))
	if err != nil {
		return nil, err
	}

	containerURL := azblob.NewContainerURL(*url, azblob.NewPipeline(credential, azblob.PipelineOptions{}))

	result := &azContext{
		container: containerURL,
		prefix:    prefix,
	}

	return result, nil
}

func (az *azContext) blobPath(path string) string {
	return filepath.Join(az.prefix, path)
}

func (az *azContext) blobURL(path string) azblob.BlobURL {
	return az.container.NewBlobURL(az.blobPath(path))
}

func (az *azContext) internalFilelist(prefix string, progress aptly.Progress) (paths []string, md5s []string, err error) {
	const delimiter = "/"
	paths = make([]string, 0, 1024)
	md5s = make([]string, 0, 1024)
	prefix = filepath.Join(az.prefix, prefix)
	if prefix != "" {
		prefix += delimiter
	}

	for marker := (azblob.Marker{}); marker.NotDone(); {
		listBlob, err := az.container.ListBlobsFlatSegment(
			context.Background(), marker, azblob.ListBlobsSegmentOptions{
				Prefix:     prefix,
				MaxResults: 1,
				Details:    azblob.BlobListingDetails{Metadata: true}})
		if err != nil {
			return nil, nil, fmt.Errorf("error listing under prefix %s in %s: %s", prefix, az, err)
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

		if progress != nil {
			time.Sleep(time.Duration(500) * time.Millisecond)
			progress.AddBar(1)
		}
	}

	return paths, md5s, nil
}

func (az *azContext) putFile(blob azblob.BlobURL, source io.Reader, sourceMD5 string) error {
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
		blob.ToBlockBlobURL(),
		uploadOptions,
	)

	return err
}

// String
func (az *azContext) String() string {
	return fmt.Sprintf("Azure: %s/%s", az.container, az.prefix)
}
