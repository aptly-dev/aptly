package deb

import (
	"archive/tar"
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/h2non/filetype/matchers"
	ar "github.com/mkrautz/goar"
	"github.com/pkg/errors"

	"github.com/aptly-dev/aptly/pgp"
	"github.com/kjk/lzma"
	"github.com/smira/go-xz"
)

// Source kinds
const (
	SourceSnapshot   = "snapshot"
	SourceLocalRepo  = "local"
	SourceRemoteRepo = "repo"
)

type parseQuery func(string) (PackageQuery, error)

// GetControlFileFromDeb reads control file from deb package
func GetControlFileFromDeb(packageFile string) (Stanza, error) {
	file, err := os.Open(packageFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	library := ar.NewReader(file)
	for {
		header, err := library.Next()

		if err == io.EOF {
			return nil, fmt.Errorf("unable to find control.tar.* part in package %s", packageFile)
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read .deb archive %s: %s", packageFile, err)
		}

		// As per deb(5) version 1.19.0.4 the control file may be:
		// - control.tar (since 1.17.6)
		// - control.tar.gz
		// - control.tar.xz (since 1.17.6)
		// Look for all of the above and uncompress as necessary.
		if strings.HasPrefix(header.Name, "control.tar") {
			bufReader := bufio.NewReader(library)

			var tarInput io.Reader

			switch header.Name {
			case "control.tar":
				tarInput = bufReader
			case "control.tar.gz":
				ungzip, err := gzip.NewReader(bufReader)
				if err != nil {
					return nil, errors.Wrapf(err, "unable to ungzip %s from %s", header.Name, packageFile)
				}
				defer ungzip.Close()
				tarInput = ungzip
			case "control.tar.xz":
				unxz, err := xz.NewReader(bufReader)
				if err != nil {
					return nil, errors.Wrapf(err, "unable to unxz %s from %s", header.Name, packageFile)
				}
				defer unxz.Close()
				tarInput = unxz
			default:
				return nil, fmt.Errorf("unsupported tar compression in %s: %s", packageFile, header.Name)
			}

			untar := tar.NewReader(tarInput)
			for {
				tarHeader, err := untar.Next()
				if err == io.EOF {
					return nil, fmt.Errorf("unable to find control file in %s", packageFile)
				}
				if err != nil {
					return nil, fmt.Errorf("unable to read .tar archive from %s. Error: %s", packageFile, err)
				}

				if tarHeader.Name == "./control" || tarHeader.Name == "control" {
					reader := NewControlFileReader(untar, false, false)
					stanza, err := reader.ReadStanza()
					if err != nil {
						return nil, err
					}

					return stanza, nil
				}
			}
		}
	}
}

// GetControlFileFromDsc reads control file from dsc package
func GetControlFileFromDsc(dscFile string, verifier pgp.Verifier) (Stanza, error) {
	file, err := os.Open(dscFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	isClearSigned, err := verifier.IsClearSigned(file)
	file.Seek(0, 0)

	if err != nil {
		return nil, err
	}

	var text io.ReadCloser

	if isClearSigned {
		text, err = verifier.ExtractClearsigned(file)
		if err != nil {
			return nil, err
		}
		defer text.Close()
	} else {
		text = file
	}

	reader := NewControlFileReader(text, false, false)
	stanza, err := reader.ReadStanza()
	if err != nil {
		return nil, err
	}

	return stanza, nil

}

// GetContentsFromDeb returns list of files installed by .deb package
func GetContentsFromDeb(file io.Reader, packageFile string) ([]string, error) {
	library := ar.NewReader(file)
	for {
		header, err := library.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("unable to find data.tar.* part in %s", packageFile)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read .deb archive from %s", packageFile)
		}

		if strings.HasPrefix(header.Name, "data.tar") {
			bufReader := bufio.NewReader(library)
			signature, err := bufReader.Peek(270)

			var isTar bool
			if err == nil {
				isTar = matchers.Tar(signature)
			}

			var tarInput io.Reader

			switch header.Name {
			case "data.tar":
				tarInput = bufReader
			case "data.tar.gz":
				if isTar {
					tarInput = bufReader
				} else {
					ungzip, err := gzip.NewReader(bufReader)
					if err != nil {
						return nil, errors.Wrapf(err, "unable to ungzip data.tar.gz from %s", packageFile)
					}
					defer ungzip.Close()
					tarInput = ungzip
				}
			case "data.tar.bz2":
				tarInput = bzip2.NewReader(bufReader)
			case "data.tar.xz":
				unxz, err := xz.NewReader(bufReader)
				if err != nil {
					return nil, errors.Wrapf(err, "unable to unxz data.tar.xz from %s", packageFile)
				}
				defer unxz.Close()
				tarInput = unxz
			case "data.tar.lzma":
				unlzma := lzma.NewReader(bufReader)
				defer unlzma.Close()
				tarInput = unlzma
			default:
				return nil, fmt.Errorf("unsupported tar compression in %s: %s", packageFile, header.Name)
			}

			untar := tar.NewReader(tarInput)
			var results []string
			for {
				tarHeader, err := untar.Next()
				if err == io.EOF {
					return results, nil
				}
				if err != nil {
					return nil, errors.Wrapf(err, "unable to read .tar archive from %s", packageFile)
				}

				if tarHeader.Typeflag == tar.TypeDir {
					continue
				}

				tarHeader.Name = strings.TrimPrefix(tarHeader.Name[2:], "./")
				results = append(results, tarHeader.Name)
			}
		}
	}
}
