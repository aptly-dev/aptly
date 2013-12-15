package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

// Downloader is parallel HTTP fetcher
type Downloader struct {
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
func NewDownloader(threads int) (downloader *Downloader) {
	downloader = &Downloader{
		queue:   make(chan *downloadTask, 1000),
		stop:    make(chan bool),
		stopped: make(chan bool),
		threads: threads,
	}

	for i := 0; i < downloader.threads; i++ {
		go downloader.process()
	}

	return
}

// Shutdown stops downloader after current tasks are finished,
// but doesn't process rest of queue
func (downloader *Downloader) Shutdown() {
	for i := 0; i < downloader.threads; i++ {
		downloader.stop <- true
	}

	for i := 0; i < downloader.threads; i++ {
		<-downloader.stopped
	}
}

// Download starts new download task
func (downloader *Downloader) Download(url string, destination string) <-chan error {
	ch := make(chan error, 1)

	downloader.queue <- &downloadTask{url: url, destination: destination, result: ch}

	return ch
}

// DownloadTemp starts new download to temporary file and returns File
//
// Temporary file would be already removed, so no need to cleanup
func (downloader *Downloader) DownloadTemp(url string) (*os.File, error) {
	ch := make(chan error, 1)

	tempfile, err := ioutil.TempFile(os.TempDir(), "aptly")
	if err != nil {
		return nil, err
	}

	defer os.Remove(tempfile.Name())

	downloader.queue <- &downloadTask{url: url, destination: tempfile.Name(), result: ch}

	err = <-ch
	if err != nil {
		tempfile.Close()
		return nil, err
	}

	return tempfile, nil
}

// handleTask processes single download task
func (downloader *Downloader) handleTask(task *downloadTask) {
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

	outfile, err := os.Create(task.destination)
	if err != nil {
		task.result <- err
		return
	}
	defer outfile.Close()

	io.Copy(outfile, resp.Body)

	task.result <- nil
}

// process implements download thread in goroutine
func (downloader *Downloader) process() {
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
