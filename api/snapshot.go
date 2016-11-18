package api

import (
	"fmt"

	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/deb"
	"github.com/gin-gonic/gin"
)

// GET /api/snapshots
func apiSnapshotsList(c *gin.Context) {
	SortMethodString := c.Request.URL.Query().Get("sort")

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()

	if SortMethodString == "" {
		SortMethodString = "name"
	}

	result := []*deb.Snapshot{}
	collection.ForEachSorted(SortMethodString, func(snapshot *deb.Snapshot) error {
		result = append(result, snapshot)
		return nil
	})

	c.JSON(200, result)
}

// POST /api/mirrors/:name/snapshots/
func apiSnapshotsCreateFromMirror(c *gin.Context) {
	var (
		err      error
		repo     *deb.RemoteRepo
		snapshot *deb.Snapshot
	)

	var b struct {
		Name        string `binding:"required"`
		Description string
	}

	if c.Bind(&b) != nil {
		return
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.RemoteRepoCollection()
	snapshotCollection := collectionFactory.SnapshotCollection()

	repo, err = collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	err = repo.CheckLock()
	if err != nil {
		c.AbortWithError(409, err)
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	snapshot, err = deb.NewSnapshotFromRepository(b.Name, repo)
	if err != nil {
		c.AbortWithError(400, err)
		return
	}

	if b.Description != "" {
		snapshot.Description = b.Description
	}

	err = snapshotCollection.Add(snapshot)
	if err != nil {
		c.AbortWithError(400, err)
		return
	}

	c.JSON(201, snapshot)
}

// POST /api/snapshots
func apiSnapshotsCreate(c *gin.Context) {
	var (
		err      error
		snapshot *deb.Snapshot
	)

	var b struct {
		Name            string `binding:"required"`
		Description     string
		SourceSnapshots []string
		PackageRefs     []string
	}

	if c.Bind(&b) != nil {
		return
	}

	if b.Description == "" {
		if len(b.SourceSnapshots)+len(b.PackageRefs) == 0 {
			b.Description = "Created as empty"
		}
	}

	collectionFactory := context.NewCollectionFactory()
	snapshotCollection := collectionFactory.SnapshotCollection()

	sources := make([]*deb.Snapshot, len(b.SourceSnapshots))

	for i := range b.SourceSnapshots {
		sources[i], err = snapshotCollection.ByName(b.SourceSnapshots[i])
		if err != nil {
			c.AbortWithError(404, err)
			return
		}

		err = snapshotCollection.LoadComplete(sources[i])
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
	}

	list := deb.NewPackageList()

	// verify package refs and build package list
	for _, ref := range b.PackageRefs {
		var p *deb.Package

		p, err = collectionFactory.PackageCollection().ByKey([]byte(ref))
		if err != nil {
			if err == database.ErrNotFound {
				c.AbortWithError(404, fmt.Errorf("package %s: %s", ref, err))
			} else {
				c.AbortWithError(500, err)
			}
			return
		}
		err = list.Add(p)
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
	}

	snapshot = deb.NewSnapshotFromRefList(b.Name, sources, deb.NewPackageRefListFromPackageList(list), b.Description)

	err = snapshotCollection.Add(snapshot)
	if err != nil {
		c.AbortWithError(400, err)
		return
	}

	c.JSON(201, snapshot)
}

// POST /api/repos/:name/snapshots
func apiSnapshotsCreateFromRepository(c *gin.Context) {
	var (
		err      error
		repo     *deb.LocalRepo
		snapshot *deb.Snapshot
	)

	var b struct {
		Name        string `binding:"required"`
		Description string
	}

	if c.Bind(&b) != nil {
		return
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()
	snapshotCollection := collectionFactory.SnapshotCollection()

	repo, err = collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	snapshot, err = deb.NewSnapshotFromLocalRepo(b.Name, repo)
	if err != nil {
		c.AbortWithError(400, err)
		return
	}

	if b.Description != "" {
		snapshot.Description = b.Description
	}

	err = snapshotCollection.Add(snapshot)
	if err != nil {
		c.AbortWithError(400, err)
		return
	}

	c.JSON(201, snapshot)
}

// PUT /api/snapshots/:name
func apiSnapshotsUpdate(c *gin.Context) {
	var (
		err      error
		snapshot *deb.Snapshot
	)

	var b struct {
		Name        string
		Description string
	}

	if c.Bind(&b) != nil {
		return
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()

	snapshot, err = collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	_, err = collection.ByName(b.Name)
	if err == nil {
		c.AbortWithError(409, fmt.Errorf("unable to rename: snapshot %s already exists", b.Name))
		return
	}

	if b.Name != "" {
		snapshot.Name = b.Name
	}

	if b.Description != "" {
		snapshot.Description = b.Description
	}

	err = collectionFactory.SnapshotCollection().Update(snapshot)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, snapshot)
}

// GET /api/snapshots/:name
func apiSnapshotsShow(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()

	snapshot, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	err = collection.LoadComplete(snapshot)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, snapshot)
}

// DELETE /api/snapshots/:name
func apiSnapshotsDrop(c *gin.Context) {
	name := c.Params.ByName("name")
	force := c.Request.URL.Query().Get("force") == "1"

	collectionFactory := context.NewCollectionFactory()
	snapshotCollection := collectionFactory.SnapshotCollection()
	publishedCollection := collectionFactory.PublishedRepoCollection()

	snapshot, err := snapshotCollection.ByName(name)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	published := publishedCollection.BySnapshot(snapshot)

	if len(published) > 0 {
		c.AbortWithError(409, fmt.Errorf("unable to drop: snapshot is published"))
		return
	}

	if !force {
		snapshots := snapshotCollection.BySnapshotSource(snapshot)
		if len(snapshots) > 0 {
			c.AbortWithError(409, fmt.Errorf("won't delete snapshot that was used as source for other snapshots, use ?force=1 to override"))
			return
		}
	}

	err = snapshotCollection.Drop(snapshot)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, gin.H{})
}

// GET /api/snapshots/:name/diff/:withSnapshot
func apiSnapshotsDiff(c *gin.Context) {
	onlyMatching := c.Request.URL.Query().Get("onlyMatching") == "1"

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()

	snapshotA, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	snapshotB, err := collection.ByName(c.Params.ByName("withSnapshot"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	err = collection.LoadComplete(snapshotA)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	err = collection.LoadComplete(snapshotB)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	// Calculate diff
	diff, err := snapshotA.RefList().Diff(snapshotB.RefList(), collectionFactory.PackageCollection())
	if err != nil {
		c.AbortWithError(500, err)
		return
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
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()

	snapshot, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	err = collection.LoadComplete(snapshot)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	showPackages(c, snapshot.RefList(), collectionFactory)
}
