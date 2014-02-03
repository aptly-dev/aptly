package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type expectedRequest struct {
	URL      string
	Err      error
	Response string
}

// FakeDownloader is like Downloader, but it used in tests
// to stub out results
type FakeDownloader struct {
	expected []expectedRequest
}

// Check interface
var (
	_ Downloader = &FakeDownloader{}
)

// NewFakeDownloader creates new expected downloader
func NewFakeDownloader() *FakeDownloader {
	result := &FakeDownloader{}
	result.expected = make([]expectedRequest, 0)
	return result
}

// ExpectResponse installs expectation on upcoming download with response
func (f *FakeDownloader) ExpectResponse(url string, response string) *FakeDownloader {
	f.expected = append(f.expected, expectedRequest{URL: url, Response: response})
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

// DownloadWithChecksum performs fake download by matching against first expectation in the queue, with cheksum verification
func (f *FakeDownloader) DownloadWithChecksum(url string, filename string, result chan<- error, expected ChecksumInfo) {
	if len(f.expected) == 0 || f.expected[0].URL != url {
		result <- fmt.Errorf("unexpected request for %s", url)
		return
	}

	expectation := f.expected[0]
	f.expected = f.expected[1:]

	if expectation.Err != nil {
		result <- expectation.Err
		return
	}

	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		result <- err
		return
	}

	outfile, err := os.Create(filename)
	if err != nil {
		result <- err
		return
	}
	defer outfile.Close()

	cks := NewChecksumWriter()
	w := io.MultiWriter(outfile, cks)

	_, err = w.Write([]byte(expectation.Response))
	if err != nil {
		result <- err
		return
	}

	if expected.MD5 != "" {
		if expected != cks.Sum() {
			result <- fmt.Errorf("checksums don't match: %#v != %#v", expected, cks.Sum())
			return
		}
	}

	result <- nil
	return
}

// Download performs fake download by matching against first expectation in the queue
func (f *FakeDownloader) Download(url string, filename string, result chan<- error) {
	f.DownloadWithChecksum(url, filename, result, ChecksumInfo{})
}

// Shutdown does nothing
func (f *FakeDownloader) Shutdown() {
}

// Pause does nothing
func (f *FakeDownloader) Pause() {
}

// Resume does nothing
func (f *FakeDownloader) Resume() {
}
