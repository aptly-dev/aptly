package http

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
)

// DownloadTemp starts new download to temporary file and returns File
//
// Temporary file would be already removed, so no need to cleanup
func DownloadTemp(downloader aptly.Downloader, url string) (*os.File, error) {
	return DownloadTempWithChecksum(downloader, url, nil, false, 1)
}

// DownloadTempWithChecksum is a DownloadTemp with checksum verification
//
// Temporary file would be already removed, so no need to cleanup
func DownloadTempWithChecksum(downloader aptly.Downloader, url string, expected *utils.ChecksumInfo, ignoreMismatch bool, maxTries int) (*os.File, error) {
	tempdir, err := ioutil.TempDir(os.TempDir(), "aptly")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempdir)

	tempfile := filepath.Join(tempdir, "buffer")

	if expected != nil && downloader.GetProgress() != nil {
		downloader.GetProgress().InitBar(expected.Size, true)
		defer downloader.GetProgress().ShutdownBar()
	}

	err = downloader.DownloadWithChecksum(url, tempfile, expected, ignoreMismatch, maxTries)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(tempfile)
	if err != nil {
		return nil, err
	}

	return file, nil
}
