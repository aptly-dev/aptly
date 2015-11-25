package deb

import (
	"archive/tar"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"github.com/mkrautz/goar"
	"github.com/smira/aptly/utils"
	"github.com/smira/go-xz"
	"github.com/smira/lzma"
	"io"
	"os"
	"strings"
)

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
			return nil, fmt.Errorf("unable to find control.tar.gz part in package %s", packageFile)
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read .deb archive %s: %s", packageFile, err)
		}

		if header.Name == "control.tar.gz" {
			ungzip, err := gzip.NewReader(library)
			if err != nil {
				return nil, fmt.Errorf("unable to ungzip control file from %s. Error: %s", packageFile, err)
			}
			defer ungzip.Close()

			untar := tar.NewReader(ungzip)
			for {
				tarHeader, err := untar.Next()
				if err == io.EOF {
					return nil, fmt.Errorf("unable to find control file in %s", packageFile)
				}
				if err != nil {
					return nil, fmt.Errorf("unable to read .tar archive from %s. Error: %s", packageFile, err)
				}

				if tarHeader.Name == "./control" || tarHeader.Name == "control" {
					reader := NewControlFileReader(untar)
					stanza, err := reader.ReadStanza(false)
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
func GetControlFileFromDsc(dscFile string, verifier utils.Verifier) (Stanza, error) {
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

	var text *os.File

	if isClearSigned {
		text, err = verifier.ExtractClearsigned(file)
		if err != nil {
			return nil, err
		}
		defer text.Close()
	} else {
		text = file
	}

	reader := NewControlFileReader(text)
	stanza, err := reader.ReadStanza(false)
	if err != nil {
		return nil, err
	}

	return stanza, nil

}

// GetContentsFromDeb returns list of files installed by .deb package
func GetContentsFromDeb(packageFile string) ([]string, error) {
	file, err := os.Open(packageFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	library := ar.NewReader(file)
	for {
		header, err := library.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("unable to find data.tar.* part in %s", packageFile)
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read .deb archive from %s: %s", packageFile, err)
		}

		if strings.HasPrefix(header.Name, "data.tar") {
			var tarInput io.Reader

			switch header.Name {
			case "data.tar":
				tarInput = library
			case "data.tar.gz":
				ungzip, err := gzip.NewReader(library)
				if err != nil {
					return nil, fmt.Errorf("unable to ungzip data.tar.gz from %s: %s", packageFile,err)
				}
				defer ungzip.Close()
				tarInput = ungzip
			case "data.tar.bz2":
				tarInput = bzip2.NewReader(library)
			case "data.tar.xz":
				unxz, err := xz.NewReader(library)
				if err != nil {
					return nil, fmt.Errorf("unable to unxz data.tar.xz from %s: %s", packageFile, err)
				}
				defer unxz.Close()
				tarInput = unxz
			case "data.tar.lzma":
				unlzma := lzma.NewReader(library)
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
					return nil, fmt.Errorf("unable to read .tar archive from %s: %s", packageFile, err)
				}

				if tarHeader.Typeflag == tar.TypeDir {
					continue
				}

				if strings.HasPrefix(tarHeader.Name, "./") {
					tarHeader.Name = tarHeader.Name[2:]
				}
				results = append(results, tarHeader.Name)
			}
		}
	}
}
