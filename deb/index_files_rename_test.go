package deb

import (
	"fmt"
	"sync"
	"time"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/utils"

	. "gopkg.in/check.v1"
)

type RenameFilesSuite struct {
	savedRenameConcurrency int
}

var _ = Suite(&RenameFilesSuite{})

func (s *RenameFilesSuite) SetUpTest(c *C) {
	s.savedRenameConcurrency = utils.Config.PublishRenameConcurrency
}

func (s *RenameFilesSuite) TearDownTest(c *C) {
	utils.Config.PublishRenameConcurrency = s.savedRenameConcurrency
}

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

	// With waitFor set, every rename blocks until that many run at once, so a
	// test proves the limit is reached rather than only respected.
	waitFor  int
	released bool
	release  chan struct{}
}

func (s *renameRecordingStorage) RenameFile(oldName, newName string) error {
	s.mu.Lock()
	s.inFlight++
	if s.inFlight > s.maxSeen {
		s.maxSeen = s.inFlight
	}
	if s.waitFor > 0 && s.inFlight == s.waitFor && !s.released {
		s.released = true
		close(s.release)
	}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.inFlight--
		s.mu.Unlock()
	}()

	if s.waitFor > 0 {
		select {
		case <-s.release:
		case <-time.After(5 * time.Second):
			return fmt.Errorf("never saw %d renames in flight at once", s.waitFor)
		}
	}

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
		rec = &renameRecordingStorage{renamed: make(map[string]string), release: make(chan struct{})}
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

func (s *RenameFilesSuite) TestRenameFilesRenamesEveryEntry(c *C) {
	files, rec := newRenameFiles(nil, 20)

	c.Assert(files.RenameFiles(), IsNil)

	c.Check(len(rec.renamed), Equals, 20)
	for i := 0; i < 20; i++ {
		c.Check(rec.renamed[fmt.Sprintf("dists/x/Packages.tmp.%d", i)], Equals,
			fmt.Sprintf("dists/x/Packages.%d", i))
	}
}

func (s *RenameFilesSuite) TestRenameFilesRunsUpToTheConfiguredLimit(c *C) {
	// Above the default of 4, so a limit that ignored the config would time out.
	utils.Config.PublishRenameConcurrency = 6
	files, rec := newRenameFiles(nil, 50)
	rec.waitFor = 6

	c.Assert(files.RenameFiles(), IsNil)

	c.Check(rec.maxSeen, Equals, 6)
	c.Check(len(rec.renamed), Equals, 50)
}

func (s *RenameFilesSuite) TestRenameFilesIsSerialAtALimitOfOneOrLess(c *C) {
	for _, limit := range []int{1, 0, -3} {
		utils.Config.PublishRenameConcurrency = limit
		files, rec := newRenameFiles(nil, 10)

		c.Assert(files.RenameFiles(), IsNil, Commentf("limit %d", limit))

		c.Check(rec.maxSeen, Equals, 1, Commentf("limit %d", limit))
		c.Check(len(rec.renamed), Equals, 10, Commentf("limit %d", limit))
	}
}

func (s *RenameFilesSuite) TestRenameFilesReportsAFailure(c *C) {
	files, rec := newRenameFiles(nil, 10)
	rec.failOn = "dists/x/Packages.tmp.4"

	err := files.RenameFiles()

	c.Assert(err, NotNil)
	c.Check(err, ErrorMatches, "unable to rename: rename refused for dists/x/Packages.tmp.4")
	// The failure is reported, and it does not abandon the rest: every other
	// entry is still renamed rather than left staged.
	c.Check(len(rec.renamed), Equals, 9)
}

func (s *RenameFilesSuite) TestRenameFilesWithNothingToDo(c *C) {
	files, _ := newRenameFiles(nil, 0)

	c.Assert(files.RenameFiles(), IsNil)
}
