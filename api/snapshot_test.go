package api

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	ctx "github.com/aptly-dev/aptly/context"
	"github.com/aptly-dev/aptly/utils"
	"github.com/smira/flag"

	"github.com/aptly-dev/aptly/deb"
	. "gopkg.in/check.v1"
)

type SnapshotsSuite struct {
	APISuite
}

var _ = Suite(&SnapshotsSuite{})

func (s *SnapshotsSuite) TestGetSnapshotsIncludesNumPackages(c *C) {
	collection := s.context.NewCollectionFactory().SnapshotCollection()
	snapshot := deb.NewSnapshotFromRefList("count-snapshot-list", nil, makePackageRefList(c), "")
	c.Assert(collection.Add(snapshot), IsNil)

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
	collection := s.context.NewCollectionFactory().SnapshotCollection()
	snapshot := deb.NewSnapshotFromRefList("broken-snapshot-list", nil, makePackageRefList(c), "")
	c.Assert(collection.Add(snapshot), IsNil)
	putRawDBValue(c, &s.APISuite, snapshot.RefKey(), []byte("not-msgpack"))

	response, err := s.HTTPRequest("GET", "/api/snapshots", nil)
	c.Assert(err, IsNil)
	c.Assert(response.Code, Equals, 500)
	c.Assert(response.Body.String(), Matches, ".*msgpack.*|.*decode.*")
}

// Pull must load the destination references before inferring architectures or
// preserving existing packages. Both fixtures are ordinary amd64 packages.
func (*SnapshotsSuite) TestPullPreservesDestinationPackages(c *C) {
	savedConfig, savedContext := utils.Config, context
	defer func() { utils.Config, context = savedConfig, savedContext }()
	dir := c.MkDir()
	configPath := filepath.Join(dir, "aptly.conf")
	configData, err := json.Marshal(map[string]interface{}{"rootDir": dir, "architectures": []string{}})
	c.Assert(err, IsNil)
	c.Assert(os.WriteFile(configPath, configData, 0600), IsNil)
	flags := flag.NewFlagSet("pull-test", flag.ContinueOnError)
	flags.String("config", configPath, "")
	flags.String("architectures", "", "")
	flags.Bool("no-lock", false, "")
	flags.Int("db-open-attempts", 1, "")
	testContext, err := ctx.NewContext(flags)
	c.Assert(err, IsNil)
	defer testContext.Shutdown()
	s := &APISuite{context: testContext, router: Router(testContext)}
	factory := s.context.NewCollectionFactory()
	for _, mode := range []string{"inferred", "explicit"} {
		a := &deb.Package{Name: "package-a", Version: "1", Architecture: "amd64"}
		b := &deb.Package{Name: "package-b", Version: "1", Architecture: "amd64"}
		for name, pkg := range map[string]*deb.Package{"base": a, "source": b} {
			c.Assert(factory.PackageCollection().Update(pkg), IsNil)
			list := deb.NewPackageList()
			c.Assert(list.Add(pkg), IsNil)
			snapshot := deb.NewSnapshotFromPackageList("pull-"+name+"-"+mode, nil, list, "")
			c.Assert(factory.SnapshotCollection().Add(snapshot), IsNil)
		}
		params := snapshotsPullParams{
			Source:      "pull-source-" + mode,
			Destination: "pull-result-" + mode,
			Queries:     []string{"package-b"},
		}
		if mode == "explicit" {
			params.Architectures = []string{"amd64"}
		}
		body, err := json.Marshal(params)
		c.Assert(err, IsNil)
		response, err := s.HTTPRequest("POST", "/api/snapshots/pull-base-"+mode+"/pull?no-remove=1&no-deps=1", bytes.NewReader(body))
		c.Assert(err, IsNil)
		if !c.Check(response.Code, Equals, 201, Commentf("%s: %s", mode, response.Body.String())) {
			continue
		}
		snapshots := s.context.NewCollectionFactory().SnapshotCollection()
		result, err := snapshots.ByName(params.Destination)
		c.Assert(err, IsNil)
		c.Assert(snapshots.LoadComplete(result), IsNil)
		c.Check(result.RefList().Strings(), DeepEquals, []string{
			string(a.Key("")), string(b.Key("")),
		}, Commentf("%s: preserve A and add B", mode))
	}
}
