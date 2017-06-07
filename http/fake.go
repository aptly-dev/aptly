package http

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
)

type expectedRequest struct {
	URL      string
	Err      error
	Response string
}

// FakeDownloader is like Downloader, but it used in tests
// to stub out results
type FakeDownloader struct {
	expected    []expectedRequest
	anyExpected map[string]expectedRequest
}

// Check interface
var (
	_ aptly.Downloader = (*FakeDownloader)(nil)
)

// NewFakeDownloader creates new expected downloader
func NewFakeDownloader() *FakeDownloader {
	result := &FakeDownloader{}
	result.expected = make([]expectedRequest, 0)
	result.anyExpected = make(map[string]expectedRequest)
	return result
}

// ExpectResponse installs expectation on upcoming download with response
func (f *FakeDownloader) ExpectResponse(url string, response string) *FakeDownloader {
	f.expected = append(f.expected, expectedRequest{URL: url, Response: response})
	return f
}

// AnyExpectResponse installs expectation on upcoming download with response in any order (url should be unique)
func (f *FakeDownloader) AnyExpectResponse(url string, response string) *FakeDownloader {
	f.anyExpected[url] = expectedRequest{URL: url, Response: response}
	return f
}

// ExpectError installs expectation on upcoming download with error
func (f *FakeDownloader) ExpectError(url string, err error) *FakeDownloader {
	f.expected = append(f.expected, expectedRequest{URL: url, Err: err})
	return f
}

// Empty verifies that are planned downloads have happened
func (f *FakeDownloader) Empty() bool {
	return len(f.expected) == 0
}

// DownloadWithChecksum performs fake download by matching against first expectation in the queue or any expectation, with cheksum verification
func (f *FakeDownloader) DownloadWithChecksum(url string, filename string, expected *utils.ChecksumInfo, ignoreMismatch bool, maxTries int) error {
	var expectation expectedRequest
	if len(f.expected) > 0 && f.expected[0].URL == url {
		expectation, f.expected = f.expected[0], f.expected[1:]
	} else if _, ok := f.anyExpected[url]; ok {
		expectation = f.anyExpected[url]
		delete(f.anyExpected, url)
	} else {
		return fmt.Errorf("unexpected request for %s", url)
	}

	if expectation.Err != nil {
		return expectation.Err
	}

	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return err
	}

	outfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outfile.Close()

	cks := utils.NewChecksumWriter()
	w := io.MultiWriter(outfile, cks)

	_, err = w.Write([]byte(expectation.Response))
	if err != nil {
		return err
	}

	if expected != nil {
		if expected.Size != cks.Sum().Size || expected.MD5 != "" && expected.MD5 != cks.Sum().MD5 ||
			expected.SHA1 != "" && expected.SHA1 != cks.Sum().SHA1 || expected.SHA256 != "" && expected.SHA256 != cks.Sum().SHA256 {
			if ignoreMismatch {
				fmt.Printf("WARNING: checksums don't match: %#v != %#v for %s\n", expected, cks.Sum(), url)
			} else {
				return fmt.Errorf("checksums don't match: %#v != %#v for %s", expected, cks.Sum(), url)
			}
		}
	}

	return nil
}

// Download performs fake download by matching against first expectation in the queue
func (f *FakeDownloader) Download(url string, filename string) error {
	return f.DownloadWithChecksum(url, filename, nil, false, 1)
}

// GetProgress returns Progress object
func (f *FakeDownloader) GetProgress() aptly.Progress {
	return nil
}
