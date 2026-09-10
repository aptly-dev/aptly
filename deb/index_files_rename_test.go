package deb

import (
	"fmt"
	"sync"

	"github.com/aptly-dev/aptly/aptly"

	. "gopkg.in/check.v1"
)

type IndexFilesSuite struct{}

var _ = Suite(&IndexFilesSuite{})

// Only RenameFile is exercised by RenameFiles, so the rest of the interface is
// embedded and left nil: a call to anything else would panic, which is the
// loudest way to notice this fake drifting from what the code under test does.
type renameRecordingStorage struct {
	aptly.PublishedStorage

	mu       sync.Mutex
	renamed  map[string]string
	failOn   string
	inFlight int
	maxSeen  int
}

func (s *renameRecordingStorage) RenameFile(oldName, newName string) error {
	s.mu.Lock()
	s.inFlight++
	if s.inFlight > s.maxSeen {
		s.maxSeen = s.inFlight
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.inFlight--
		s.mu.Unlock()
	}()

	if oldName == s.failOn {
		return fmt.Errorf("rename refused for %s", oldName)
	}

	s.mu.Lock()
	s.renamed[oldName] = newName
	s.mu.Unlock()
	return nil
}

func newRenameFiles(storage aptly.PublishedStorage, n int) (*indexFiles, *renameRecordingStorage) {
	rec, ok := storage.(*renameRecordingStorage)
	if !ok {
		rec = &renameRecordingStorage{renamed: make(map[string]string)}
	}
	files := &indexFiles{
		publishedStorage: rec,
		renameMap:        make(map[string]string, n),
	}
	for i := 0; i < n; i++ {
		files.renameMap[fmt.Sprintf("dists/x/Packages.tmp.%d", i)] = fmt.Sprintf("dists/x/Packages.%d", i)
	}
	return files, rec
}

func (s *IndexFilesSuite) TestRenameFilesRenamesEveryEntry(c *C) {
	files, rec := newRenameFiles(nil, 20)

	c.Assert(files.RenameFiles(), IsNil)

	c.Check(len(rec.renamed), Equals, 20)
	for i := 0; i < 20; i++ {
		c.Check(rec.renamed[fmt.Sprintf("dists/x/Packages.tmp.%d", i)], Equals,
			fmt.Sprintf("dists/x/Packages.%d", i))
	}
}

func (s *IndexFilesSuite) TestRenameFilesStaysWithinTheConcurrencyLimit(c *C) {
	files, rec := newRenameFiles(nil, 50)

	c.Assert(files.RenameFiles(), IsNil)

	// An upper bound, so this cannot flake: fewer in flight than the limit is
	// always acceptable, more never is.
	c.Check(rec.maxSeen <= renameConcurrency, Equals, true,
		Commentf("saw %d renames in flight, limit is %d", rec.maxSeen, renameConcurrency))
}

func (s *IndexFilesSuite) TestRenameFilesReportsAFailure(c *C) {
	files, rec := newRenameFiles(nil, 10)
	rec.failOn = "dists/x/Packages.tmp.4"

	err := files.RenameFiles()

	c.Assert(err, NotNil)
	c.Check(err, ErrorMatches, "unable to rename: rename refused for dists/x/Packages.tmp.4")
	// The failure is reported, and it does not abandon the rest: every other
	// entry is still renamed rather than left staged.
	c.Check(len(rec.renamed), Equals, 9)
}

func (s *IndexFilesSuite) TestRenameFilesWithNothingToDo(c *C) {
	files, _ := newRenameFiles(nil, 0)

	c.Assert(files.RenameFiles(), IsNil)
}
