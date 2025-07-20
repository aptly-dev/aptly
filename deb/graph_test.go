package deb

import (
	"github.com/aptly-dev/aptly/database/goleveldb"
	. "gopkg.in/check.v1"
)

type GraphSuite struct {
	collectionFactory *CollectionFactory
}

var _ = Suite(&GraphSuite{})

func (s *GraphSuite) SetUpTest(c *C) {
	db, _ := goleveldb.NewOpenDB(c.MkDir())
	s.collectionFactory = NewCollectionFactory(db)
}

func (s *GraphSuite) TearDownTest(c *C) {
	// Collections are closed automatically when the test ends
}

func (s *GraphSuite) TestBuildGraphBasic(c *C) {
	// Test BuildGraph with default (horizontal) layout
	graph, err := BuildGraph(s.collectionFactory, "horizontal")
	c.Check(err, IsNil)
	c.Check(graph, NotNil)
}

func (s *GraphSuite) TestBuildGraphVertical(c *C) {
	// Test BuildGraph with vertical layout
	graph, err := BuildGraph(s.collectionFactory, "vertical")
	c.Check(err, IsNil)
	c.Check(graph, NotNil)
}

func (s *GraphSuite) TestBuildGraphUnknownLayout(c *C) {
	// Test BuildGraph with unknown layout (should default to horizontal)
	graph, err := BuildGraph(s.collectionFactory, "unknown")
	c.Check(err, IsNil)
	c.Check(graph, NotNil)
}
