package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mxk/go-flowrate/flowrate"
	"github.com/pkg/errors"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
	"github.com/smira/go-ftp-protocol/protocol"
)

// Check interface
var (
	_ aptly.Downloader = (*downloaderImpl)(nil)
)

// downloaderImpl is implementation of Downloader interface
type downloaderImpl struct {
	progress  aptly.Progress
	aggWriter io.Writer
	client    *http.Client
}

// NewDownloader creates new instance of Downloader which specified number
// of threads and download limit in bytes/sec
func NewDownloader(downLimit int64, progress aptly.Progress) aptly.Downloader {
	transport := http.Transport{}
	transport.Proxy = http.DefaultTransport.(*http.Transport).Proxy
	transport.ResponseHeaderTimeout = 30 * time.Second
	transport.TLSHandshakeTimeout = http.DefaultTransport.(*http.Transport).TLSHandshakeTimeout
	transport.ExpectContinueTimeout = http.DefaultTransport.(*http.Transport).ExpectContinueTimeout
	transport.DisableCompression = true
	initTransport(&transport)
	transport.RegisterProtocol("ftp", &protocol.FTPRoundTripper{})

	downloader := &downloaderImpl{
		progress: progress,
		client: &http.Client{
			Transport: &transport,
		},
	}

	if downLimit > 0 {
		downloader.aggWriter = flowrate.NewWriter(progress, downLimit)
	} else {
		downloader.aggWriter = progress
	}

	return downloader
}

// GetProgress returns Progress object
func (downloader *downloaderImpl) GetProgress() aptly.Progress {
	return downloader.progress
}

// Download starts new download task
func (downloader *downloaderImpl) Download(url string, destination string) error {
	return downloader.DownloadWithChecksum(url, destination, nil, false, 1)
}

// DownloadWithChecksum starts new download task with checksum verification
func (downloader *downloaderImpl) DownloadWithChecksum(url string, destination string,
	expected *utils.ChecksumInfo, ignoreMismatch bool, maxTries int) error {

	downloader.progress.Printf("Downloading %s...\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return errors.Wrap(err, url)
	}
	req.Close = true

	proxyURL, _ := downloader.client.Transport.(*http.Transport).Proxy(req)
	if proxyURL == nil && (req.URL.Scheme == "http" || req.URL.Scheme == "https") {
		req.URL.Opaque = strings.Replace(req.URL.RequestURI(), "+", "%2b", -1)
		req.URL.RawQuery = ""
	}

	var temppath string
	for maxTries > 0 {
		temppath, err = downloader.download(req, url, destination, expected, ignoreMismatch)

		if err != nil {
			maxTries--
		} else {
			// successful download
			break
		}
	}

	// still an error after retrying, giving up
	if err != nil {
		return err
	}

	err = os.Rename(temppath, destination)
	if err != nil {
		os.Remove(temppath)
		return errors.Wrap(err, url)
	}

	return nil
}

func (downloader *downloaderImpl) download(req *http.Request, url, destination string, expected *utils.ChecksumInfo, ignoreMismatch bool) (string, error) {
	resp, err := downloader.client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, url)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", &Error{Code: resp.StatusCode, URL: url}
	}

	err = os.MkdirAll(filepath.Dir(destination), 0777)
	if err != nil {
		return "", errors.Wrap(err, url)
	}

	temppath := destination + ".down"

	outfile, err := os.Create(temppath)
	if err != nil {
		return "", errors.Wrap(err, url)
	}
	defer outfile.Close()

	checksummer := utils.NewChecksumWriter()
	writers := []io.Writer{outfile, downloader.aggWriter}

	if expected != nil {
		writers = append(writers, checksummer)
	}

	w := io.MultiWriter(writers...)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		os.Remove(temppath)
		return "", errors.Wrap(err, url)
	}

	if expected != nil {
		actual := checksummer.Sum()

		if actual.Size != expected.Size {
			err = fmt.Errorf("%s: size check mismatch %d != %d", url, actual.Size, expected.Size)
		} else if expected.MD5 != "" && actual.MD5 != expected.MD5 {
			err = fmt.Errorf("%s: md5 hash mismatch %#v != %#v", url, actual.MD5, expected.MD5)
		} else if expected.SHA1 != "" && actual.SHA1 != expected.SHA1 {
			err = fmt.Errorf("%s: sha1 hash mismatch %#v != %#v", url, actual.SHA1, expected.SHA1)
		} else if expected.SHA256 != "" && actual.SHA256 != expected.SHA256 {
			err = fmt.Errorf("%s: sha256 hash mismatch %#v != %#v", url, actual.SHA256, expected.SHA256)
		} else if expected.SHA512 != "" && actual.SHA512 != expected.SHA512 {
			err = fmt.Errorf("%s: sha512 hash mismatch %#v != %#v", url, actual.SHA512, expected.SHA512)
		}

		if err != nil {
			if ignoreMismatch {
				downloader.progress.Printf("WARNING: %s\n", err.Error())
			} else {
				os.Remove(temppath)
				return "", err
			}
		} else {
			// update checksums if they match, so that they contain exactly expected set
			*expected = actual
		}
	}

	return temppath, nil
}
