package azure

// Package azure handles publishing to Azure Storage

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/aptly-dev/aptly/aptly"
)

func isBlobNotFound(err error) bool {
	var respErr *azcore.ResponseError
	if errors.As(err, &respErr) {
		return respErr.StatusCode == 404 // BlobNotFound
	}
	return false
}

type azContext struct {
	client    *azblob.Client
	container string
	prefix    string
}

func newAzContext(accountName, accountKey, container, prefix, endpoint string) (*azContext, error) {
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, err
	}

	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.blob.core.windows.net", accountName)
	}

	serviceClient, err := azblob.NewClientWithSharedKeyCredential(endpoint, cred, nil)
	if err != nil {
		return nil, err
	}

	result := &azContext{
		client:    serviceClient,
		container: container,
		prefix:    prefix,
	}

	return result, nil
}

func (az *azContext) blobPath(path string) string {
	return filepath.Join(az.prefix, path)
}

func (az *azContext) internalFilelist(prefix string, progress aptly.Progress) (paths []string, md5s []string, err error) {
	const delimiter = "/"
	paths = make([]string, 0, 1024)
	md5s = make([]string, 0, 1024)
	prefix = filepath.Join(az.prefix, prefix)
	if prefix != "" {
		prefix += delimiter
	}

	ctx := context.Background()
	maxResults := int32(1)
	pager := az.client.NewListBlobsFlatPager(az.container, &azblob.ListBlobsFlatOptions{
		Prefix:     &prefix,
		MaxResults: &maxResults,
		Include:    azblob.ListBlobsInclude{Metadata: true},
	})

	// Iterate over each page
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("error listing under prefix %s in %s: %s", prefix, az, err)
		}

		for _, blob := range page.Segment.BlobItems {
			if prefix == "" {
				paths = append(paths, *blob.Name)
			} else {
				name := *blob.Name
				paths = append(paths, name[len(prefix):])
			}
			b := *blob
			md5 := b.Properties.ContentMD5
			md5s = append(md5s, fmt.Sprintf("%x", md5))

		}
		if progress != nil {
			time.Sleep(time.Duration(500) * time.Millisecond)
			progress.AddBar(1)
		}
	}

	return paths, md5s, nil
}

func (az *azContext) putFile(blobName string, source io.Reader, sourceMD5 string) error {
	uploadOptions := &azblob.UploadFileOptions{
		BlockSize:   4 * 1024 * 1024,
		Concurrency: 8,
	}

	path := az.blobPath(blobName)
	if len(sourceMD5) > 0 {
		decodedMD5, err := hex.DecodeString(sourceMD5)
		if err != nil {
			return err
		}
		uploadOptions.HTTPHeaders = &blob.HTTPHeaders{
			BlobContentMD5: decodedMD5,
		}
	}

	var err error
	if file, ok := source.(*os.File); ok {
		_, err = az.client.UploadFile(context.TODO(), az.container, path, file, uploadOptions)
	}

	return err
}

// String
func (az *azContext) String() string {
	return fmt.Sprintf("Azure: %s/%s", az.container, az.prefix)
}
