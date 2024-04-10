package http

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"github.com/aptly-dev/aptly/utils"
	"github.com/cavaliergopher/grab/v3"
	"github.com/pkg/errors"

	"github.com/aptly-dev/aptly/aptly"
)

type GrabDownloader struct {
	client    *grab.Client
	progress  aptly.Progress
	maxTries  int
	downLimit int64
}

// Check interface
var (
	_ aptly.Downloader = (*GrabDownloader)(nil)
)

// NewGrabDownloader creates new expected downloader
func NewGrabDownloader(downLimit int64, maxTries int, progress aptly.Progress) *GrabDownloader {
	client := grab.NewClient()
	return &GrabDownloader{
		client:    client,
		progress:  progress,
		maxTries:  maxTries,
		downLimit: downLimit,
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
		d.log("Download Error: %v\n", err)
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
	fmt.Printf(msg, a...)
	if d.progress != nil {
		d.progress.Printf(msg, a...)
	}
}

func (d *GrabDownloader) maybeSetupChecksum(req *grab.Request, expected *utils.ChecksumInfo) error {
	if expected == nil {
		// Nothing to setup
		return nil
	}
	if expected.MD5 != "" {
		expectedHash, err := hex.DecodeString(expected.MD5)
		if err != nil {
			return err
		}
		req.SetChecksum(md5.New(), expectedHash, true)
	} else if expected.SHA1 != "" {
		expectedHash, err := hex.DecodeString(expected.SHA1)
		if err != nil {
			return err
		}
		req.SetChecksum(sha1.New(), expectedHash, true)
	} else if expected.SHA256 != "" {
		expectedHash, err := hex.DecodeString(expected.SHA256)
		if err != nil {
			return err
		}
		req.SetChecksum(sha256.New(), expectedHash, true)
	} else if expected.SHA512 != "" {
		expectedHash, err := hex.DecodeString(expected.SHA512)
		if err != nil {
			return err
		}
		req.SetChecksum(sha512.New(), expectedHash, true)
	}
	req.Size = expected.Size
	return nil
}

func (d *GrabDownloader) download(_ context.Context, url string, destination string, expected *utils.ChecksumInfo, ignoreMismatch bool) error {
	// TODO clean up dest dir on permanent failure
	d.log("Download %s -> %s\n", url, destination)

	req, err := grab.NewRequest(destination, url)
	if err != nil {
		d.log("Error creating new request: %v\n", err)
		return errors.Wrap(err, url)
	}
	if d.downLimit > 0 {
		req.RateLimiter = rate.NewLimiter(rate.Limit(d.downLimit), int(d.downLimit))
	}

	d.maybeSetupChecksum(req, expected)
	if err != nil {
		d.log("Error setting up checksum: %v\n", err)
		return errors.Wrap(err, url)
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
	err = resp.Err()
	if err != nil && err == grab.ErrBadChecksum && ignoreMismatch {
		fmt.Printf("Ignoring checksum mismatch for %s\n", url)
		return nil
	}
	return err
}

func (d *GrabDownloader) GetProgress() aptly.Progress {
	return d.progress
}

func (d *GrabDownloader) GetLength(_ context.Context, url string) (int64, error) {
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
