package http

import (
	"compress/bzip2"
	"compress/gzip"
	"context"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
	xz "github.com/smira/go-xz"
)

// List of extensions + corresponding uncompression support
var compressionMethods = []struct {
	extenstion     string
	transformation func(io.Reader) (io.Reader, error)
}{
	{
		extenstion:     ".bz2",
		transformation: func(r io.Reader) (io.Reader, error) { return bzip2.NewReader(r), nil },
	},
	{
		extenstion:     ".gz",
		transformation: func(r io.Reader) (io.Reader, error) { return gzip.NewReader(r) },
	},
	{
		extenstion:     ".xz",
		transformation: func(r io.Reader) (io.Reader, error) { return xz.NewReader(r) },
	},
	{
		extenstion:     "",
		transformation: func(r io.Reader) (io.Reader, error) { return r, nil },
	},
}

// DownloadTryCompression tries to download from URL .bz2, .gz and raw extension until
// it finds existing file.
func DownloadTryCompression(ctx context.Context, downloader aptly.Downloader, baseURL *url.URL, path string, expectedChecksums map[string]utils.ChecksumInfo, ignoreMismatch bool) (io.Reader, *os.File, error) {
	var err error

	for _, method := range compressionMethods {
		var file *os.File

		tryPath := path + method.extenstion
		foundChecksum := false

		bestSuffix := ""

		for suffix := range expectedChecksums {
			if strings.HasSuffix(tryPath, suffix) {
				foundChecksum = true
				if len(suffix) > len(bestSuffix) {
					bestSuffix = suffix
				}
			}
		}

		tryURL := baseURL.ResolveReference(&url.URL{Path: tryPath})

		if foundChecksum {
			expected := expectedChecksums[bestSuffix]
			file, err = DownloadTempWithChecksum(ctx, downloader, tryURL.String(), &expected, ignoreMismatch)
		} else {
			if !ignoreMismatch {
				continue
			}

			file, err = DownloadTemp(ctx, downloader, tryURL.String())
		}

		if err != nil {
			if err1, ok := err.(*Error); ok && (err1.Code == 404 || err1.Code == 403) {
				continue
			}
			return nil, nil, err
		}

		var uncompressed io.Reader
		uncompressed, err = method.transformation(file)
		if err != nil {
			return nil, nil, err
		}

		return uncompressed, file, err
	}

	if err == nil {
		return nil, nil, &NoCandidateFoundError{URL: baseURL.ResolveReference(&url.URL{Path: path})}
	}
	return nil, nil, err
}
