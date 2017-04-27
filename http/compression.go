package http

import (
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
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
func DownloadTryCompression(downloader aptly.Downloader, url string, expectedChecksums map[string]utils.ChecksumInfo, ignoreMismatch bool, maxTries int) (io.Reader, *os.File, error) {
	var err error

	for _, method := range compressionMethods {
		var file *os.File

		tryURL := url + method.extenstion
		foundChecksum := false

		for suffix, expected := range expectedChecksums {
			if strings.HasSuffix(tryURL, suffix) {
				file, err = DownloadTempWithChecksum(downloader, tryURL, &expected, ignoreMismatch, maxTries)
				foundChecksum = true
				break
			}
		}

		if !foundChecksum {
			if !ignoreMismatch {
				continue
			}

			file, err = DownloadTemp(downloader, tryURL)
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
		err = fmt.Errorf("no candidates for %s found", url)
	}
	return nil, nil, err
}
