package http

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"
	"github.com/mxk/go-flowrate/flowrate"
	"github.com/pkg/errors"
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
	maxTries  int
	client    *http.Client
}

// NewDownloader creates new instance of Downloader which specified number
// of threads and download limit in bytes/sec
func NewDownloader(downLimit int64, maxTries int, progress aptly.Progress) aptly.Downloader {
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
		maxTries: maxTries,
		client: &http.Client{
			Transport: &transport,
		},
	}

	progressWriter := io.Writer(progress)
	if progress == nil {
		progressWriter = ioutil.Discard
	}

	downloader.client.CheckRedirect = downloader.checkRedirect
	if downLimit > 0 {
		downloader.aggWriter = flowrate.NewWriter(progressWriter, downLimit)
	} else {
		downloader.aggWriter = progressWriter
	}

	return downloader
}

func (downloader *downloaderImpl) checkRedirect(req *http.Request, via []*http.Request) error {
	if downloader.progress != nil {
		downloader.progress.Printf("Following redirect to %s...\n", req.URL)
	}

	return nil
}

// GetProgress returns Progress object
func (downloader *downloaderImpl) GetProgress() aptly.Progress {
	return downloader.progress
}

// GetLength of given url
func (downloader *downloaderImpl) GetLength(ctx context.Context, url string) (int64, error) {
	req, err := downloader.newRequest(ctx, "HEAD", url)
	if err != nil {
		return -1, err
	}

	var resp *http.Response

	maxTries := downloader.maxTries
	for maxTries > 0 {
		resp, err = downloader.client.Do(req)
		if err != nil && retryableError(err) {
			maxTries--
		} else {
			// stop retrying
			break
		}
	}

	if err != nil {
		return -1, errors.Wrap(err, url)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return -1, &Error{Code: resp.StatusCode, URL: url}
	}

	if resp.ContentLength < 0 {
		return -1, fmt.Errorf("could not determine length of %s", url)
	}

	return resp.ContentLength, nil
}

// Download starts new download task
func (downloader *downloaderImpl) Download(ctx context.Context, url string, destination string) error {
	return downloader.DownloadWithChecksum(ctx, url, destination, nil, false)
}

func retryableError(err error) bool {
	// unwrap errors.Wrap
	err = errors.Cause(err)

	// unwrap *url.Error
	if wrapped, ok := err.(*url.Error); ok {
		err = wrapped.Err
	}

	switch err {
	case io.EOF:
		return true
	case io.ErrUnexpectedEOF:
		return true
	}

	switch err.(type) {
	case *net.OpError:
		return true
	case syscall.Errno:
		return true
	case net.Error:
		return true
	}
	// Note: make all errors retryable
	return true
}

func (downloader *downloaderImpl) newRequest(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return nil, errors.Wrap(err, url)
	}
	req.Close = true
	req = req.WithContext(ctx)

	proxyURL, _ := downloader.client.Transport.(*http.Transport).Proxy(req)
	if proxyURL == nil && (req.URL.Scheme == "http" || req.URL.Scheme == "https") {
		req.URL.Opaque = strings.Replace(req.URL.RequestURI(), "+", "%2b", -1)
		req.URL.RawQuery = ""
	}

	return req, nil
}

// DownloadWithChecksum starts new download task with checksum verification
func (downloader *downloaderImpl) DownloadWithChecksum(ctx context.Context, url string, destination string,
	expected *utils.ChecksumInfo, ignoreMismatch bool) error {

	if downloader.progress != nil {
		downloader.progress.Printf("Downloading %s...\n", url)
	}
	req, err := downloader.newRequest(ctx, "GET", url)

	var temppath string
	maxTries := downloader.maxTries
	const delayBase = 1
	const delayMultiplier = 2
	delay := time.Duration(delayBase * time.Second)
	for maxTries > 0 {
		temppath, err = downloader.download(req, url, destination, expected, ignoreMismatch)

		if err != nil {
			if retryableError(err) {
				if downloader.progress != nil {
					downloader.progress.Printf("Error downloading %s: %s retrying...\n", url, err)
				}
				maxTries--
				time.Sleep(delay)
				// Sleep exponentially at the next retry
				delay *= delayMultiplier
			} else {
				if downloader.progress != nil {
					downloader.progress.Printf("Error downloading %s: %s cannot retry...\n", url, err)
				}
				break
			}
		} else {
			// get out of the loop
			if downloader.progress != nil {
				downloader.progress.Printf("Success downloading %s\n", url)
			}
			break
		}
		if downloader.progress != nil {
			downloader.progress.Printf("Retrying %d %s...\n", maxTries, url)
		}
	}

	// still an error after retrying, giving up
	if err != nil {
		if downloader.progress != nil {
			downloader.progress.Printf("Giving up on %s...\n", url)
		}
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
	writers := []io.Writer{outfile}

	if downloader.progress != nil {
		writers = append(writers, downloader.progress)
	}

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
				if downloader.progress != nil {
					downloader.progress.Printf("WARNING: %s\n", err.Error())
				}
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
