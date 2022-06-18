package utils

import (
	"io"
	"os"
	"os/exec"

	"github.com/klauspost/pgzip"
)

// CompressFile compresses file specified by source to .gz & .bz2
//
// It uses internal gzip and external bzip2, see:
// https://code.google.com/p/go/issues/detail?id=4828
func CompressFile(source *os.File, onlyGzip bool) error {
	gzPath := source.Name() + ".gz"
	gzFile, err := os.Create(gzPath)
	if err != nil {
		return err
	}
	defer gzFile.Close()

	gzWriter := pgzip.NewWriter(gzFile)
	defer gzWriter.Close()

	source.Seek(0, 0)
	_, err = io.Copy(gzWriter, source)
	if err != nil || onlyGzip {
		return err
	}

	cmd := exec.Command("bzip2", "-k", "-f", source.Name())
	return cmd.Run()
}
