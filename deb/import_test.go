package deb

import (
	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/database/goleveldb"

	. "gopkg.in/check.v1"
)

type ImportSuite struct {
	db                database.Storage
	packageCollection *PackageCollection
	reporter          *aptly.RecordingResultReporter
}

var _ = Suite(&ImportSuite{})

func (s *ImportSuite) SetUpTest(c *C) {
	s.db, _ = goleveldb.NewOpenDB(c.MkDir())
	s.packageCollection = NewPackageCollection(s.db)
	s.reporter = &aptly.RecordingResultReporter{
		Warnings:     []string{},
		AddedLines:   []string{},
		RemovedLines: []string{},
	}
}

func (s *ImportSuite) TearDownTest(c *C) {
	_ = s.db.Close()
}

func (s *ImportSuite) TestImportPackageFilesRejectsInvalidVersion(c *C) {
	list := NewPackageList()

	processedFiles, failedFiles, err := ImportPackageFiles(
		list, []string{"testdata/import/invalid-version.dsc"}, false, &NullVerifier{},
		nil, s.packageCollection, s.reporter, nil,
		func(database.ReaderWriter) aptly.ChecksumStorage { return nil })

	c.Assert(err, IsNil)
	c.Check(processedFiles, HasLen, 0)
	c.Check(failedFiles, DeepEquals, []string{"testdata/import/invalid-version.dsc"})
	c.Check(s.reporter.Warnings, DeepEquals, []string{
		"Version number ('1.2.3-') for the 'aptly-test-invalid-version' package is invalid",
	})
}
