package api

import (
	"fmt"
	"sort"
	"github.com/gin-gonic/gin"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/query"
)

// GET /api/snapshots
func apiSnapshotsList(c *gin.Context) {
	SortMethodString := c.Request.URL.Query().Get("sort")

	collection := context.CollectionFactory().SnapshotCollection()
	collection.Sort(SortMethodString)

	result := []*deb.Snapshot{}
	collection.ForEach(func(snapshot *deb.Snapshot) error {
		result = append(result, snapshot)
		return nil
	})

	c.JSON(200, result)
}

// POST /api/mirros/:name/snapshots/:snapname
func apiSnapshotsCreateFromMirror(c *gin.Context) {
	var (
		err       error
		repo     *deb.RemoteRepo
		snapshot *deb.Snapshot
	)

	var b struct {
		Name                string `binding:"required"`
	}

	if !c.Bind(&b) {
		return
	}

	collection := context.CollectionFactory().RemoteRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	snapshotCollection := context.CollectionFactory().SnapshotCollection()
	snapshotCollection.Lock()
	defer snapshotCollection.Unlock()

	repo, err = collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	err = repo.CheckLock()
	if err != nil {
		c.Fail(409, err)
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.Fail(500, err)
		return
	}

	snapshot, err = deb.NewSnapshotFromRepository(b.Name, repo)
	if err != nil {
		c.Fail(500, err)
		return
	}

	err = snapshotCollection.Add(snapshot)
	if err != nil {
		c.Fail(500, err)
		return
	}

	c.JSON(201, snapshot)
}

// POST /api/repos/:name/snapshots/:snapname
func apiSnapshotsCreateEmpty(c *gin.Context) {
	var (
		err       error
		snapshot *deb.Snapshot
	)

	var b struct {
		Name                string `binding:"required"`
	}

	if !c.Bind(&b) {
		return
	}

	snapshotCollection := context.CollectionFactory().SnapshotCollection()
	snapshotCollection.Lock()
	defer snapshotCollection.Unlock()

	packageList := deb.NewPackageList()

	snapshot = deb.NewSnapshotFromPackageList(b.Name, nil, packageList, "Created as empty")

	err = snapshotCollection.Add(snapshot)
	if err != nil {
		c.Fail(500, err)
		return
	}

	c.JSON(201, snapshot)
}

// POST /api/repos/:name/snapshots/:snapname
func apiSnapshotsCreateFromRepository(c *gin.Context) {
	var (
		err       error
		repo     *deb.LocalRepo
		snapshot *deb.Snapshot
	)

	var b struct {
		Name                string `binding:"required"`
	}

	if !c.Bind(&b) {
		return
	}

	collection := context.CollectionFactory().LocalRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	snapshotCollection := context.CollectionFactory().SnapshotCollection()
	snapshotCollection.Lock()
	defer snapshotCollection.Unlock()

	repo, err = collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.Fail(500, err)
		return
	}

	snapshot, err = deb.NewSnapshotFromLocalRepo(b.Name, repo)
	if err != nil {
		c.Fail(500, err)
		return
	}

	err = snapshotCollection.Add(snapshot)
	if err != nil {
		c.Fail(500, err)
		return
	}

	c.JSON(201, snapshot)
}

// PUT /api/snapshots/:name
func apiSnapshotsRename(c *gin.Context) {
	var (
		err      error
		snapshot *deb.Snapshot
	)

	var b struct {
		Name    string `binding:"required"`
	}

	if !c.Bind(&b) {
		return
	}

	collection := context.CollectionFactory().SnapshotCollection()
	collection.Lock()
	defer collection.Unlock()

	snapshot, err = context.CollectionFactory().SnapshotCollection().ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
	}

	_, err = context.CollectionFactory().SnapshotCollection().ByName(b.Name)
	if err == nil {
		c.Fail(409, fmt.Errorf("unable to rename: snapshot %s already exists", b.Name))
	}

	snapshot.Name = b.Name
	err = context.CollectionFactory().SnapshotCollection().Update(snapshot)
	if err != nil {
		c.Fail(403, err)
	}

	c.JSON(200, snapshot)
}

// GET /api/snapshots/:name
func apiSnapshotsShow(c *gin.Context) {
	collection := context.CollectionFactory().SnapshotCollection()
	collection.RLock()
	defer collection.RUnlock()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	c.JSON(200, repo)
}

// DELETE /api/snapshots/:name
func apiSnapshotsDrop(c *gin.Context) {
	name := c.Params.ByName("name")
	force := c.Request.URL.Query().Get("force") == "1"

	collection := context.CollectionFactory().LocalRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	snapshotCollection := context.CollectionFactory().SnapshotCollection()
	snapshotCollection.RLock()
	defer snapshotCollection.RUnlock()

	publishedCollection := context.CollectionFactory().PublishedRepoCollection()
	publishedCollection.RLock()
	defer publishedCollection.RUnlock()

	snapshot, err := snapshotCollection.ByName(name)
	if err != nil {
		c.Fail(404, err)
		return
	}

	published := publishedCollection.BySnapshot(snapshot)

	if len(published) > 0 {
		for _, repo := range published {
			err = publishedCollection.LoadComplete(repo, context.CollectionFactory())
			if err != nil {
				c.Fail(409, err)
				return
			}
		}

		c.Fail(409, fmt.Errorf("unable to drop: snapshot is published"))
		return
	}

	if !force {
		snapshots := snapshotCollection.BySnapshotSource(snapshot)
		if len(snapshots) > 0 {
			c.Fail(409, fmt.Errorf("won't delete snapshot that was used as source for other snapshots, use ?force=1 to override"))
			return
		}
	}

	err = context.CollectionFactory().SnapshotCollection().Drop(snapshot)
	if err != nil {
		c.Fail(409, err)
		return
	}

	c.JSON(200, gin.H{})
}

// GET /api/snapshots/:name
func apiSnapshotsDiff(c *gin.Context) {
	onlyMatching := c.Request.URL.Query().Get("onlyMatching") == "1"

	collection := context.CollectionFactory().SnapshotCollection()
	collection.RLock()
	defer collection.RUnlock()

	snapshotA, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	snapshotB, err := collection.ByName(c.Params.ByName("withSnapshot"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	err = context.CollectionFactory().SnapshotCollection().LoadComplete(snapshotA)
	if err != nil {
		c.Fail(500, err)
	}

	err = context.CollectionFactory().SnapshotCollection().LoadComplete(snapshotB)
	if err != nil {
		c.Fail(500, err)
	}

	// Calculate diff
	diff, err := snapshotA.RefList().Diff(snapshotB.RefList(), context.CollectionFactory().PackageCollection())
	if err != nil {
		c.Fail(500, err)
	}

	result := []deb.PackageDiff{}

	for _, pdiff := range diff {
		if onlyMatching && (pdiff.Left == nil || pdiff.Right == nil) {
			continue
		}

		result = append(result, pdiff)
	}

	c.JSON(200, result)
}

// GET /api/snapshots/:name/packages
func apiSnapshotsSearchPackages(c *gin.Context) {
	collection := context.CollectionFactory().SnapshotCollection()
	collection.RLock()
	defer collection.RUnlock()

	snapshot, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	err = collection.LoadComplete(snapshot)
	if err != nil {
		c.Fail(400, err)
		return
	}

	reflist := snapshot.RefList()
	result := []*deb.Package{}

	list, err := deb.NewPackageListFromRefList(reflist, context.CollectionFactory().PackageCollection(), context.Progress())
	if err != nil {
		c.Fail(400, err)
		return
	}

	queryS := c.Request.URL.Query().Get("q")
	if queryS != "" {
		q, err := query.Parse(c.Request.URL.Query().Get("q"))
		if err != nil {
			c.Fail(401, err)
			return
		}

		withDeps := c.Request.URL.Query().Get("withDeps") == "1"
		architecturesList := []string{}

		if withDeps {
			if len(context.ArchitecturesList()) > 0 {
				architecturesList = context.ArchitecturesList()
			} else {
				architecturesList = list.Architectures(false)
			}

			sort.Strings(architecturesList)

			if len(architecturesList) == 0 {
				c.Fail(400, fmt.Errorf("unable to determine list of architectures, please specify explicitly"))
				return
			}
		}

		list.PrepareIndex()

		packages, err := list.Filter([]deb.PackageQuery{q}, withDeps,
			nil, context.DependencyOptions(), architecturesList)
		if err != nil {
			c.Fail(400, fmt.Errorf("unable to search: %s", err))
		}

		packages.ForEach(func(p *deb.Package) error {
			result = append(result, p)
			return nil
		})
	} else {
		list.ForEach(func(p *deb.Package) error {
			result = append(result, p)
			return nil
		})
	}

	c.JSON(200, result)
}
