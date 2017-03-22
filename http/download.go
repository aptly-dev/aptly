package http

import (
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mxk/go-flowrate/flowrate"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
	"github.com/smira/go-ftp-protocol/protocol"
	"github.com/smira/go-xz"
)

// Error is download error connected to HTTP code
type Error struct {
	Code int
	URL  string
}

// Error
func (e *Error) Error() string {
	return fmt.Sprintf("HTTP code %d while fetching %s", e.Code, e.URL)
}

// Check interface
var (
	_ aptly.Downloader = (*downloaderImpl)(nil)
)

// downloaderImpl is implementation of Downloader interface
type downloaderImpl struct {
	queue     chan *downloadTask
	stop      chan struct{}
	stopped   chan struct{}
	pause     chan struct{}
	unpause   chan struct{}
	progress  aptly.Progress
	aggWriter io.Writer
	threads   int
	client    *http.Client
}

// downloadTask represents single item in queue
type downloadTask struct {
	url            string
	destination    string
	result         chan<- error
	expected       utils.ChecksumInfo
	ignoreMismatch bool
	triesLeft      int
}

// NewDownloader creates new instance of Downloader which specified number
// of threads and download limit in bytes/sec
func NewDownloader(threads int, downLimit int64, progress aptly.Progress) aptly.Downloader {
	transport := http.Transport{}
	transport.Proxy = http.DefaultTransport.(*http.Transport).Proxy
	transport.ResponseHeaderTimeout = 30 * time.Second
	transport.TLSHandshakeTimeout = http.DefaultTransport.(*http.Transport).TLSHandshakeTimeout
	transport.ExpectContinueTimeout = http.DefaultTransport.(*http.Transport).ExpectContinueTimeout
	transport.DisableCompression = true
	initTransport(&transport)
	transport.RegisterProtocol("ftp", &protocol.FTPRoundTripper{})

	downloader := &downloaderImpl{
		queue:    make(chan *downloadTask, 1000),
		stop:     make(chan struct{}, threads),
		stopped:  make(chan struct{}, threads),
		pause:    make(chan struct{}),
		unpause:  make(chan struct{}),
		threads:  threads,
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

	for i := 0; i < downloader.threads; i++ {
		go downloader.process()
	}

	return downloader
}

// Shutdown stops downloader after current tasks are finished,
// but doesn't process rest of queue
func (downloader *downloaderImpl) Shutdown() {
	for i := 0; i < downloader.threads; i++ {
		downloader.stop <- struct{}{}
	}

	for i := 0; i < downloader.threads; i++ {
		<-downloader.stopped
	}
}

// Abort stops downloader but doesn't wait for downloader to stop
func (downloader *downloaderImpl) Abort() {
	for i := 0; i < downloader.threads; i++ {
		downloader.stop <- struct{}{}
	}
}

// Pause pauses task processing
func (downloader *downloaderImpl) Pause() {
	for i := 0; i < downloader.threads; i++ {
		downloader.pause <- struct{}{}
	}
}

// Resume resumes task processing
func (downloader *downloaderImpl) Resume() {
	for i := 0; i < downloader.threads; i++ {
		downloader.unpause <- struct{}{}
	}
}

// GetProgress returns Progress object
func (downloader *downloaderImpl) GetProgress() aptly.Progress {
	return downloader.progress
}

// Download starts new download task
func (downloader *downloaderImpl) Download(url string, destination string, result chan<- error) {
	downloader.DownloadWithChecksum(url, destination, result, utils.ChecksumInfo{Size: -1}, false, 1)
}

// DownloadWithChecksum starts new download task with checksum verification
func (downloader *downloaderImpl) DownloadWithChecksum(url string, destination string, result chan<- error,
	expected utils.ChecksumInfo, ignoreMismatch bool, maxTries int) {
	downloader.queue <- &downloadTask{url: url, destination: destination, result: result, expected: expected, ignoreMismatch: ignoreMismatch, triesLeft: maxTries}
}

// handleTask processes single download task
func (downloader *downloaderImpl) handleTask(task *downloadTask) {
	downloader.progress.Printf("Downloading %s...\n", task.url)

	req, err := http.NewRequest("GET", task.url, nil)
	if err != nil {
		task.result <- fmt.Errorf("%s: %s", task.url, err)
		return
	}
	req.Close = true

	proxyURL, _ := downloader.client.Transport.(*http.Transport).Proxy(req)
	if proxyURL == nil && (req.URL.Scheme == "http" || req.URL.Scheme == "https") {
		req.URL.Opaque = strings.Replace(req.URL.RequestURI(), "+", "%2b", -1)
		req.URL.RawQuery = ""
	}

	var temppath string
	for task.triesLeft > 0 {

		temppath, err = downloader.downloadTask(req, task)

		if err != nil {
			task.triesLeft--
		} else {
			// successful download
			break
		}
	}

	// still an error after retrying, giving up
	if err != nil {
		task.result <- err
		return
	}

	err = os.Rename(temppath, task.destination)
	if err != nil {
		os.Remove(temppath)
		task.result <- fmt.Errorf("%s: %s", task.url, err)
		return
	}

	task.result <- nil
}

func (downloader *downloaderImpl) downloadTask(req *http.Request, task *downloadTask) (string, error) {
	resp, err := downloader.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %s", task.url, err)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", &Error{Code: resp.StatusCode, URL: task.url}
	}

	err = os.MkdirAll(filepath.Dir(task.destination), 0777)
	if err != nil {
		return "", fmt.Errorf("%s: %s", task.url, err)
	}

	temppath := task.destination + ".down"

	outfile, err := os.Create(temppath)
	if err != nil {
		return "", fmt.Errorf("%s: %s", task.url, err)
	}
	defer outfile.Close()

	checksummer := utils.NewChecksumWriter()
	writers := []io.Writer{outfile, downloader.aggWriter}

	if task.expected.Size != -1 {
		writers = append(writers, checksummer)
	}

	w := io.MultiWriter(writers...)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		os.Remove(temppath)
		return "", fmt.Errorf("%s: %s", task.url, err)
	}

	if task.expected.Size != -1 {
		actual := checksummer.Sum()

		if actual.Size != task.expected.Size {
			err = fmt.Errorf("%s: size check mismatch %d != %d", task.url, actual.Size, task.expected.Size)
		} else if task.expected.MD5 != "" && actual.MD5 != task.expected.MD5 {
			err = fmt.Errorf("%s: md5 hash mismatch %#v != %#v", task.url, actual.MD5, task.expected.MD5)
		} else if task.expected.SHA1 != "" && actual.SHA1 != task.expected.SHA1 {
			err = fmt.Errorf("%s: sha1 hash mismatch %#v != %#v", task.url, actual.SHA1, task.expected.SHA1)
		} else if task.expected.SHA256 != "" && actual.SHA256 != task.expected.SHA256 {
			err = fmt.Errorf("%s: sha256 hash mismatch %#v != %#v", task.url, actual.SHA256, task.expected.SHA256)
		} else if task.expected.SHA512 != "" && actual.SHA512 != task.expected.SHA512 {
			err = fmt.Errorf("%s: sha512 hash mismatch %#v != %#v", task.url, actual.SHA512, task.expected.SHA512)
		}

		if err != nil {
			if task.ignoreMismatch {
				downloader.progress.Printf("WARNING: %s\n", err.Error())
			} else {
				os.Remove(temppath)
				return "", err
			}
		}
	}

	return temppath, nil
}

// process implements download thread in goroutine
func (downloader *downloaderImpl) process() {
	for {
		select {
		case <-downloader.stop:
			downloader.stopped <- struct{}{}
			return
		case <-downloader.pause:
			<-downloader.unpause
		case task := <-downloader.queue:
			downloader.handleTask(task)
		}
	}
}

// DownloadTemp starts new download to temporary file and returns File
//
// Temporary file would be already removed, so no need to cleanup
func DownloadTemp(downloader aptly.Downloader, url string) (*os.File, error) {
	return DownloadTempWithChecksum(downloader, url, utils.ChecksumInfo{Size: -1}, false, 1)
}

// DownloadTempWithChecksum is a DownloadTemp with checksum verification
//
// Temporary file would be already removed, so no need to cleanup
func DownloadTempWithChecksum(downloader aptly.Downloader, url string, expected utils.ChecksumInfo, ignoreMismatch bool, maxTries int) (*os.File, error) {
	tempdir, err := ioutil.TempDir(os.TempDir(), "aptly")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempdir)

	tempfile := filepath.Join(tempdir, "buffer")

	if expected.Size != -1 && downloader.GetProgress() != nil {
		downloader.GetProgress().InitBar(expected.Size, true)
		defer downloader.GetProgress().ShutdownBar()
	}

	ch := make(chan error, 1)
	downloader.DownloadWithChecksum(url, tempfile, ch, expected, ignoreMismatch, maxTries)

	err = <-ch

	if err != nil {
		return nil, err
	}

	file, err := os.Open(tempfile)
	if err != nil {
		return nil, err
	}

	return file, nil
}

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
				file, err = DownloadTempWithChecksum(downloader, tryURL, expected, ignoreMismatch, maxTries)
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
