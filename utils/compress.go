package utils

import (
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/klauspost/pgzip"
)

// CompressFile compresses file specified by source to .gz & .bz2
//
// It uses internal gzip and external bzip2, see:
// https://code.google.com/p/go/issues/detail?id=4828
func CompressFile(source *os.File, onlyGzip bool) error {
	if onlyGzip {
		return compressGzip(source.Name())
	}

	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		errs <- compressGzip(source.Name())
	}()

	go func() {
		defer wg.Done()
		cmd := exec.Command("bzip2", "-1", "-k", "-f", source.Name())
		errs <- cmd.Run()
	}()

	wg.Wait()
	close(errs)

	var firstErr error
	for err := range errs {
		if firstErr == nil && err != nil {
			firstErr = err
		}
	}
	return firstErr
}

func compressGzip(sourceFilename string) error {
	source, err := os.Open(sourceFilename)
	if err != nil {
		return err
	}
	defer func() {
		_ = source.Close()
	}()

	gzPath := sourceFilename + ".gz"
	gzFile, err := os.Create(gzPath)
	if err != nil {
		return err
	}

	gzWriter, err := pgzip.NewWriterLevel(gzFile, gzip.BestSpeed)
	if err != nil {
		_ = gzFile.Close()
		return err
	}

	_, err = io.Copy(gzWriter, source)
	if closeErr := gzWriter.Close(); err == nil {
		err = closeErr
	}
	if closeErr := gzFile.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}

	return nil
}
