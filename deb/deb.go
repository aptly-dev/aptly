package deb

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/mkrautz/goar"
	"github.com/smira/aptly/utils"
	"io"
	"os"
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
			return nil, fmt.Errorf("unable to find control.tar.gz part")
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read .deb archive: %s", err)
		}

		if header.Name == "control.tar.gz" {
			ungzip, err := gzip.NewReader(library)
			if err != nil {
				return nil, fmt.Errorf("unable to ungzip: %s", err)
			}
			defer ungzip.Close()

			untar := tar.NewReader(ungzip)
			for {
				tarHeader, err := untar.Next()
				if err == io.EOF {
					return nil, fmt.Errorf("unable to find control file")
				}
				if err != nil {
					return nil, fmt.Errorf("unable to read .tar archive: %s", err)
				}

				if tarHeader.Name == "./control" || tarHeader.Name == "control" {
					reader := NewControlFileReader(untar)
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
	stanza, err := reader.ReadStanza()
	if err != nil {
		return nil, err
	}

	return stanza, nil

}
