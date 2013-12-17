package utils

import (
	"fmt"
	"os"
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

// Download performs fake download by matching against first expectation in the queue
func (f *FakeDownloader) Download(url string, filename string) <-chan error {
	result := make(chan error, 1)

	if len(f.expected) == 0 || f.expected[0].URL != url {
		result <- fmt.Errorf("unexpected request for %s", url)
		return result
	}

	expected := f.expected[0]
	f.expected = f.expected[1:]

	if expected.Err != nil {
		result <- expected.Err
		return result
	}

	outfile, err := os.Create(filename)
	if err != nil {
		result <- err
		return result
	}
	defer outfile.Close()

	_, err = outfile.Write([]byte(expected.Response))
	if err != nil {
		result <- err
		return result
	}

	result <- nil
	return result
}

// Shutdown does nothing
func (f *FakeDownloader) Shutdown() {
}
