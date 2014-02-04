package utils

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
)

// Downloader is parallel HTTP fetcher
type Downloader interface {
	Download(url string, destination string, result chan<- error)
	DownloadWithChecksum(url string, destination string, result chan<- error, expected ChecksumInfo, ignoreMismatch bool)
	Pause()
	Resume()
	Shutdown()
	GetProgress() *Progress
}

// Check interface
var (
	_ Downloader = &downloaderImpl{}
)

// downloaderImpl is implementation of Downloader interface
type downloaderImpl struct {
	queue    chan *downloadTask
	stop     chan bool
	stopped  chan bool
	pause    chan bool
	unpause  chan bool
	progress *Progress
	threads  int
}

// downloadTask represents single item in queue
type downloadTask struct {
	url            string
	destination    string
	result         chan<- error
	expected       ChecksumInfo
	ignoreMismatch bool
}

// NewDownloader creates new instance of Downloader which specified number
// of threads
func NewDownloader(threads int) Downloader {
	downloader := &downloaderImpl{
		queue:    make(chan *downloadTask, 1000),
		stop:     make(chan bool),
		stopped:  make(chan bool),
		pause:    make(chan bool),
		unpause:  make(chan bool),
		threads:  threads,
		progress: NewProgress(),
	}

	downloader.progress.Start()

	for i := 0; i < downloader.threads; i++ {
		go downloader.process()
	}

	return downloader
}

// Shutdown stops downloader after current tasks are finished,
// but doesn't process rest of queue
func (downloader *downloaderImpl) Shutdown() {
	for i := 0; i < downloader.threads; i++ {
		downloader.stop <- true
	}

	for i := 0; i < downloader.threads; i++ {
		<-downloader.stopped
	}

	downloader.progress.Shutdown()
}

// Pause pauses task processing
func (downloader *downloaderImpl) Pause() {
	for i := 0; i < downloader.threads; i++ {
		downloader.pause <- true
	}
}

// Resume resumes task processing
func (downloader *downloaderImpl) Resume() {
	for i := 0; i < downloader.threads; i++ {
		downloader.unpause <- true
	}
}

// Resume resumes task processing
func (downloader *downloaderImpl) GetProgress() *Progress {
	return downloader.progress
}

// Download starts new download task
func (downloader *downloaderImpl) Download(url string, destination string, result chan<- error) {
	downloader.DownloadWithChecksum(url, destination, result, ChecksumInfo{Size: -1}, false)
}

// DownloadWithChecksum starts new download task with checksum verification
func (downloader *downloaderImpl) DownloadWithChecksum(url string, destination string, result chan<- error,
	expected ChecksumInfo, ignoreMismatch bool) {
	downloader.queue <- &downloadTask{url: url, destination: destination, result: result, expected: expected, ignoreMismatch: ignoreMismatch}
}

// handleTask processes single download task
func (downloader *downloaderImpl) handleTask(task *downloadTask) {
	downloader.progress.Printf("Downloading %s...\n", task.url)

	resp, err := http.Get(task.url)
	if err != nil {
		task.result <- err
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		task.result <- fmt.Errorf("HTTP code %d while fetching %s", resp.StatusCode, task.url)
		return
	}

	err = os.MkdirAll(filepath.Dir(task.destination), 0755)
	if err != nil {
		task.result <- err
		return
	}

	temppath := task.destination + ".down"

	outfile, err := os.Create(temppath)
	if err != nil {
		task.result <- err
		return
	}
	defer outfile.Close()

	checksummer := NewChecksumWriter()
	writers := []io.Writer{outfile}

	if task.expected.Size != -1 {
		writers = append(writers, checksummer)
	}

	if downloader.progress != nil {
		writers = append(writers, downloader.progress)
	}

	w := io.MultiWriter(writers...)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		os.Remove(temppath)
		task.result <- err
		return
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
		}

		if err != nil {
			if task.ignoreMismatch {
				downloader.progress.Printf("WARNING: %s\n", err.Error())
			} else {
				os.Remove(temppath)
				task.result <- err
				return
			}
		}
	}

	err = os.Rename(temppath, task.destination)
	if err != nil {
		os.Remove(temppath)
		task.result <- err
		return
	}

	task.result <- nil
}

// process implements download thread in goroutine
func (downloader *downloaderImpl) process() {
	for {
		select {
		case <-downloader.stop:
			downloader.stopped <- true
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
func DownloadTemp(downloader Downloader, url string) (*os.File, error) {
	return DownloadTempWithChecksum(downloader, url, ChecksumInfo{Size: -1}, false)
}

// DownloadTempWithChecksum is a DownloadTemp with checksum verification
//
// Temporary file would be already removed, so no need to cleanup
func DownloadTempWithChecksum(downloader Downloader, url string, expected ChecksumInfo, ignoreMismatch bool) (*os.File, error) {
	tempdir, err := ioutil.TempDir(os.TempDir(), "aptly")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempdir)

	tempfile := filepath.Join(tempdir, "buffer")

	ch := make(chan error, 1)
	downloader.DownloadWithChecksum(url, tempfile, ch, expected, ignoreMismatch)

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
		extenstion:     "",
		transformation: func(r io.Reader) (io.Reader, error) { return r, nil },
	},
}

// DownloadTryCompression tries to download from URL .bz2, .gz and raw extension until
// it finds existing file.
func DownloadTryCompression(downloader Downloader, url string, expectedChecksums map[string]ChecksumInfo, ignoreMismatch bool) (io.Reader, *os.File, error) {
	var err error

	for _, method := range compressionMethods {
		var file *os.File

		tryUrl := url + method.extenstion
		foundChecksum := false

		for suffix, expected := range expectedChecksums {
			if strings.HasSuffix(tryUrl, suffix) {
				file, err = DownloadTempWithChecksum(downloader, tryUrl, expected, ignoreMismatch)
				foundChecksum = true
				break
			}
		}

		if !foundChecksum {
			file, err = DownloadTemp(downloader, tryUrl)
		}

		if err != nil {
			continue
		}

		var uncompressed io.Reader
		uncompressed, err = method.transformation(file)
		if err != nil {
			continue
		}

		return uncompressed, file, err
	}
	return nil, nil, err
}
