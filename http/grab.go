package http

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"github.com/aptly-dev/aptly/utils"
	"github.com/cavaliercoder/grab"
	"net/http"
	"time"

	"github.com/aptly-dev/aptly/aptly"
)

type GrabDownloader struct {
	client   *grab.Client
	maxTries int
	progress aptly.Progress
}

// Check interface
var (
	_ aptly.Downloader = (*GrabDownloader)(nil)
)

// NewGrabDownloader creates new expected downloader
func NewGrabDownloader(downLimit int64, maxTries int, progress aptly.Progress) *GrabDownloader {
	// TODO rate limiting and progress
	client := grab.NewClient()
	return &GrabDownloader{
		client:   client,
		maxTries: maxTries,
	}
}

func (d *GrabDownloader) Download(ctx context.Context, url string, destination string) error {
	return d.DownloadWithChecksum(ctx, url, destination, nil, false)
}

func (d *GrabDownloader) DownloadWithChecksum(ctx context.Context, url string, destination string, expected *utils.ChecksumInfo, ignoreMismatch bool) error {
	maxTries := d.maxTries
	const delayMax = time.Duration(5 * time.Minute)
	delay := time.Duration(1 * time.Second)
	const delayMultiplier = 2
	err := fmt.Errorf("No tries available")
	for maxTries > 0 {
		err = d.download(ctx, url, destination, expected, ignoreMismatch)
		if err == nil {
			// Success
			break
		}
		d.log("Error downloading %s: %v\n", url, err)
		if retryableError(err) {
			maxTries--
			d.log("Retrying download %s: %d\n", url, maxTries)
			time.Sleep(delay)
		} else {
			// Can't retry
			d.log("Cannot retry download %s\n", url)
			break
		}
	}
	return err
}

func (d *GrabDownloader) log(msg string, a ...interface{}) {
	if d.progress != nil {
		d.progress.Printf(msg, a...)
	}
}

func (d *GrabDownloader) download(ctx context.Context, url string, destination string, expected *utils.ChecksumInfo, ignoreMismatch bool) error {
	// TODO clean up dest dir on permanent failure
	d.log("Starting download %s -> %s\n", url, destination)
	req, _ := grab.NewRequest(destination, url)

	// TODO ignoreMismatch
	if expected != nil {
		if expected.MD5 != "" {
			req.SetChecksum(md5.New(), []byte(expected.MD5), true)
		} else if expected.SHA1 != "" {
			req.SetChecksum(sha1.New(), []byte(expected.SHA1), true)
		} else if expected.SHA256 != "" {
			req.SetChecksum(sha256.New(), []byte(expected.SHA256), true)
		} else if expected.SHA512 != "" {
			req.SetChecksum(sha512.New(), []byte(expected.SHA512), true)
		}
	}

	resp := d.client.Do(req)

Loop:
	for {
		select {
		case <-resp.Done:
			// download is complete
			break Loop
		}
	}
	return resp.Err()
}

func (d *GrabDownloader) GetProgress() aptly.Progress {
	return d.progress
}

func (f *GrabDownloader) GetLength(ctx context.Context, url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return -1, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return -1, &Error{Code: resp.StatusCode, URL: url}
	}

	if resp.ContentLength < 0 {
		return -1, fmt.Errorf("could not determine length of %s", url)
	}

	return resp.ContentLength, nil
}
