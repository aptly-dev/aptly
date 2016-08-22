package utils

import (
	"compress/gzip"
	"github.com/ulikunitz/xz"
	"io"
	"os"
	"os/exec"
)

// CompressFile compresses file specified by source to .gz, .xz, and .bz2
//
// It uses internal gzip, external xz, and external bzip2, see:
// https://code.google.com/p/go/issues/detail?id=4828
func CompressFile(source *os.File) error {
	// gz compression
	gzPath := source.Name() + ".gz"
	gzFile, err := os.Create(gzPath)
	if err != nil {
		return err
	}
	defer gzFile.Close()

	gzWriter := gzip.NewWriter(gzFile)
	defer gzWriter.Close()

	source.Seek(0, 0)
	_, err = io.Copy(gzWriter, source)
	if err != nil {
		return err
	}

	// xz compression
	xzPath := source.Name() + ".xz"
	xzFile, err := os.Create(xzPath)
	if err != nil {
		return err
	}
	defer xzFile.Close()

	xzWriter := xz.NewWriter(xzFile)
	defer xzWriter.Close()

	source.Seek(0, 0)
	_, err = io.Copy(xzWriter, source)
	if err != nil {
		return err
	}

	// bzip compression
	cmd := exec.Command("bzip2", "-k", "-f", source.Name())
	return cmd.Run()
}
