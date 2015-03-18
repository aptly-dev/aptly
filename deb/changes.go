package deb

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Changes is a result of .changes file parsing
type Changes struct {
	Changes               string
	Distribution          string
	Files                 PackageFiles
	BasePath, ChangesName string
	TempDir               string
	Stanza                Stanza
}

func NewChanges(path string) (*Changes, error) {
	var err error

	c := &Changes{
		BasePath:    filepath.Dir(path),
		ChangesName: filepath.Base(path),
	}

	c.TempDir, err = ioutil.TempDir(os.TempDir(), "aptly")
	if err != nil {
		return nil, err
	}

	// copy .changes file into temporary directory
	err = utils.CopyFile(filepath.Join(c.BasePath, c.ChangesName), filepath.Join(c.TempDir, c.ChangesName))
	if err != nil {
		return nil, err
	}

	return c, nil
}

// VerifyAndParse does optional signature verification and parses changes files
func (c *Changes) VerifyAndParse(acceptUnsigned, ignoreSignature bool, verifier utils.Verifier) error {
	input, err := os.Open(filepath.Join(c.TempDir, c.ChangesName))
	if err != nil {
		return err
	}
	defer input.Close()

	isClearSigned, err := verifier.IsClearSigned(input)
	if err != nil {
		return err
	}

	input.Seek(0, 0)

	if !isClearSigned && !acceptUnsigned {
		return fmt.Errorf(".changes file is not signed and unsigned processing hasn't been enabled")
	}

	if isClearSigned && !ignoreSignature {
		_, err = verifier.VerifyClearsigned(input, false)
		if err != nil {
			return err
		}
		input.Seek(0, 0)
	}

	var text *os.File

	if isClearSigned {
		text, err = verifier.ExtractClearsigned(input)
		if err != nil {
			return err
		}
		defer text.Close()
	} else {
		text = input
	}

	reader := NewControlFileReader(text)
	c.Stanza, err = reader.ReadStanza()
	if err != nil {
		return err
	}

	c.Distribution = c.Stanza["Distribution"]
	c.Changes = c.Stanza["Changes"]

	c.Files, err = c.Files.ParseSumFields(c.Stanza)
	if err != nil {
		return err
	}

	return nil
}

// Prepare creates temporary directory, copies file there and verifies checksums
func (c *Changes) Prepare() error {
	var err error

	for _, file := range c.Files {
		if filepath.Dir(file.Filename) != "." {
			return fmt.Errorf("file is not in the same folder as .changes file: %s", file.Filename)
		}

		file.Filename = filepath.Base(file.Filename)

		err = utils.CopyFile(filepath.Join(c.BasePath, file.Filename), filepath.Join(c.TempDir, file.Filename))
		if err != nil {
			return err
		}
	}

	for _, file := range c.Files {
		var info utils.ChecksumInfo

		info, err = utils.ChecksumsForFile(filepath.Join(c.TempDir, file.Filename))
		if err != nil {
			return err
		}

		if info.Size != file.Checksums.Size {
			return fmt.Errorf("size mismatch: expected %v != obtained %v", file.Checksums.Size, info.Size)
		}

		if info.MD5 != file.Checksums.MD5 {
			return fmt.Errorf("checksum mismatch MD5: expected %v != obtained %v", file.Checksums.MD5, info.MD5)
		}

		if info.SHA1 != file.Checksums.SHA1 {
			return fmt.Errorf("checksum mismatch SHA1: expected %v != obtained %v", file.Checksums.SHA1, info.SHA1)
		}

		if info.SHA256 != file.Checksums.SHA256 {
			return fmt.Errorf("checksum mismatch SHA256 expected %v != obtained %v", file.Checksums.SHA256, info.SHA256)
		}
	}

	return nil
}

// Cleanup removes all temporary files
func (c *Changes) Cleanup() error {
	if c.TempDir == "" {
		return nil
	}

	return os.RemoveAll(c.TempDir)
}

// CollectChangesFiles walks filesystem collecting all .changes files
func CollectChangesFiles(locations []string, reporter aptly.ResultReporter) (changesFiles, failedFiles []string) {
	for _, location := range locations {
		info, err2 := os.Stat(location)
		if err2 != nil {
			reporter.Warning("Unable to process %s: %s", location, err2)
			failedFiles = append(failedFiles, location)
			continue
		}
		if info.IsDir() {
			err2 = filepath.Walk(location, func(path string, info os.FileInfo, err3 error) error {
				if err3 != nil {
					return err3
				}
				if info.IsDir() {
					return nil
				}

				if strings.HasSuffix(info.Name(), ".changes") {
					changesFiles = append(changesFiles, path)
				}

				return nil
			})
		} else if strings.HasSuffix(info.Name(), ".changes") {
			changesFiles = append(changesFiles, location)
		}
	}

	sort.Strings(changesFiles)

	return
}
