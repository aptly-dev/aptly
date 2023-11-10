package api

import (
	"encoding/json"

	"github.com/aptly-dev/aptly/deb"
	. "gopkg.in/check.v1"
)

type SnapshotsSuite struct {
	APISuite
}

var _ = Suite(&SnapshotsSuite{})

func (s *SnapshotsSuite) TestGetSnapshotsIncludesNumPackages(c *C) {
	collectionFactory := s.context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()
	reflistCollection := collectionFactory.RefListCollection()

	snapshot := deb.NewSnapshotFromRefList("count-snapshot-list", nil, makePackageRefList(c), "")
	c.Assert(collection.Add(snapshot, reflistCollection), IsNil)

	response, err := s.HTTPRequest("GET", "/api/snapshots", nil)
	c.Assert(err, IsNil)
	c.Assert(response.Code, Equals, 200)

	var snapshots []map[string]interface{}
	err = json.Unmarshal(response.Body.Bytes(), &snapshots)
	c.Assert(err, IsNil)

	found := false
	for _, snapshot := range snapshots {
		if snapshot["Name"] == "count-snapshot-list" {
			found = true
			value, ok := snapshot["NumPackages"]
			c.Assert(ok, Equals, true)
			c.Assert(value, Equals, float64(2))
			break
		}
	}

	c.Assert(found, Equals, true)
}

func (s *SnapshotsSuite) TestGetSnapshotsReturns500OnCorruptRefList(c *C) {
	collectionFactory := s.context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()
	reflistCollection := collectionFactory.RefListCollection()

	snapshot := deb.NewSnapshotFromRefList("broken-snapshot-list", nil, makePackageRefList(c), "")
	c.Assert(collection.Add(snapshot, reflistCollection), IsNil)
	putRawDBValue(c, &s.APISuite, snapshot.RefKey(), []byte("not-msgpack"))

	response, err := s.HTTPRequest("GET", "/api/snapshots", nil)
	c.Assert(err, IsNil)
	c.Assert(response.Code, Equals, 500)
	c.Assert(response.Body.String(), Matches, ".*msgpack.*|.*decode.*")
}
