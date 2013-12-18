package utils

import (
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Downloader is parallel HTTP fetcher
type Downloader interface {
	Download(url string, destination string, result chan<- error)
	Shutdown()
}

// Check interface
var (
	_ Downloader = &downloaderImpl{}
)

// downloaderImpl is implementation of Downloader interface
type downloaderImpl struct {
	queue   chan *downloadTask
	stop    chan bool
	stopped chan bool
	threads int
}

// downloadTask represents single item in queue
type downloadTask struct {
	url         string
	destination string
	result      chan<- error
}

// NewDownloader creates new instance of Downloader which specified number
// of threads
func NewDownloader(threads int) Downloader {
	downloader := &downloaderImpl{
		queue:   make(chan *downloadTask, 1000),
		stop:    make(chan bool),
		stopped: make(chan bool),
		threads: threads,
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
		downloader.stop <- true
	}

	for i := 0; i < downloader.threads; i++ {
		<-downloader.stopped
	}
}

// Download starts new download task
func (downloader *downloaderImpl) Download(url string, destination string, result chan<- error) {
	downloader.queue <- &downloadTask{url: url, destination: destination, result: result}
}

// handleTask processes single download task
func (downloader *downloaderImpl) handleTask(task *downloadTask) {
	log.Printf("Downloading %s...\n", task.url)

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

	_, err = io.Copy(outfile, resp.Body)
	if err != nil {
		os.Remove(temppath)
		task.result <- err
		return
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
		case task := <-downloader.queue:
			downloader.handleTask(task)
		}
	}
}

// DownloadTemp starts new download to temporary file and returns File
//
// Temporary file would be already removed, so no need to cleanup
func DownloadTemp(downloader Downloader, url string) (*os.File, error) {
	tempdir, err := ioutil.TempDir(os.TempDir(), "aptly")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempdir)

	tempfile := filepath.Join(tempdir, "buffer")

	ch := make(chan error, 1)
	downloader.Download(url, tempfile, ch)

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
func DownloadTryCompression(downloader Downloader, url string) (io.Reader, *os.File, error) {
	var err error

	for _, method := range compressionMethods {
		var file *os.File

		file, err = DownloadTemp(downloader, url+method.extenstion)
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
