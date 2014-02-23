package debian

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"github.com/mkrautz/goar"
	"github.com/smira/aptly/utils"
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

				if tarHeader.Name == "./control" {
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

	line, err := bufio.NewReader(file).ReadString('\n')
	if err != nil {
		return nil, err
	}

	file.Seek(0, 0)

	var text *os.File

	if strings.Index(line, "BEGIN PGP SIGN") != -1 {
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
