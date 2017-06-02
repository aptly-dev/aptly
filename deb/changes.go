package deb

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/pgp"
	"github.com/smira/aptly/utils"
)

// Changes is a result of .changes file parsing
type Changes struct {
	Changes               string
	Distribution          string
	Files                 PackageFiles
	BasePath, ChangesName string
	TempDir               string
	Source                string
	Binary                []string
	Architectures         []string
	Stanza                Stanza
	SignatureKeys         []pgp.Key
}

// NewChanges moves .changes file into temporary directory and creates Changes structure
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
func (c *Changes) VerifyAndParse(acceptUnsigned, ignoreSignature bool, verifier pgp.Verifier) error {
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
		var keyInfo *pgp.KeyInfo
		keyInfo, err = verifier.VerifyClearsigned(input, false)
		if err != nil {
			return err
		}
		input.Seek(0, 0)

		c.SignatureKeys = keyInfo.GoodKeys
	}

	var text io.ReadCloser

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
	c.Stanza, err = reader.ReadStanza(false)
	if err != nil {
		return err
	}

	c.Distribution = c.Stanza["Distribution"]
	c.Changes = c.Stanza["Changes"]
	c.Source = c.Stanza["Source"]
	c.Binary = strings.Fields(c.Stanza["Binary"])
	c.Architectures = strings.Fields(c.Stanza["Architecture"])

	c.Files, err = c.Files.ParseSumFields(c.Stanza)
	return err
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

// PackageQuery returns query that every package should match to be included
func (c *Changes) PackageQuery() (PackageQuery, error) {
	var archQuery PackageQuery = &FieldQuery{Field: "$Architecture", Relation: VersionEqual, Value: ""}
	for _, arch := range c.Architectures {
		archQuery = &OrQuery{L: &FieldQuery{Field: "$Architecture", Relation: VersionEqual, Value: arch}, R: archQuery}
	}

	// if c.Source is empty, this would never match
	sourceQuery := &AndQuery{
		L: &FieldQuery{Field: "$PackageType", Relation: VersionEqual, Value: ArchitectureSource},
		R: &FieldQuery{Field: "Name", Relation: VersionEqual, Value: c.Source},
	}

	var binaryQuery PackageQuery
	if len(c.Binary) > 0 {
		binaryQuery = &FieldQuery{Field: "Name", Relation: VersionEqual, Value: c.Binary[0]}
		// matching debug ddeb packages, they're not present in the Binary field
		var ddebQuery PackageQuery = &FieldQuery{Field: "Name", Relation: VersionEqual, Value: fmt.Sprintf("%s-dbgsym", c.Binary[0])}

		for _, binary := range c.Binary[1:] {
			binaryQuery = &OrQuery{
				L: &FieldQuery{Field: "Name", Relation: VersionEqual, Value: binary},
				R: binaryQuery,
			}
			ddebQuery = &OrQuery{
				L: &FieldQuery{Field: "Name", Relation: VersionEqual, Value: fmt.Sprintf("%s-dbgsym", binary)},
				R: ddebQuery,
			}
		}

		ddebQuery = &AndQuery{
			L: &FieldQuery{Field: "Source", Relation: VersionEqual, Value: c.Source},
			R: ddebQuery,
		}

		binaryQuery = &OrQuery{
			L: binaryQuery,
			R: ddebQuery,
		}

		binaryQuery = &AndQuery{
			L: &NotQuery{Q: &FieldQuery{Field: "$PackageType", Relation: VersionEqual, Value: ArchitectureSource}},
			R: binaryQuery}
	}

	var nameQuery PackageQuery
	if binaryQuery == nil {
		nameQuery = sourceQuery
	} else {
		nameQuery = &OrQuery{L: sourceQuery, R: binaryQuery}
	}

	return &AndQuery{L: archQuery, R: nameQuery}, nil
}

// GetField implements PackageLike interface
func (c *Changes) GetField(field string) string {
	return c.Stanza[field]
}

// MatchesDependency implements PackageLike interface
func (c *Changes) MatchesDependency(d Dependency) bool {
	return false
}

// MatchesArchitecture implements PackageLike interface
func (c *Changes) MatchesArchitecture(arch string) bool {
	return false
}

// GetName implements PackageLike interface
func (c *Changes) GetName() string {
	return ""
}

// GetVersion implements PackageLike interface
func (c *Changes) GetVersion() string {
	return ""

}

// GetArchitecture implements PackageLike interface
func (c *Changes) GetArchitecture() string {
	return ""
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

			if err2 != nil {
				reporter.Warning("Unable to process %s: %s", location, err2)
				failedFiles = append(failedFiles, location)
				continue
			}
		} else if strings.HasSuffix(info.Name(), ".changes") {
			changesFiles = append(changesFiles, location)
		}
	}

	sort.Strings(changesFiles)

	return
}
