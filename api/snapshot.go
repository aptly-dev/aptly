package api

import (
	"fmt"
	"net/http"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/task"
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
	name := c.Params.ByName("name")

	repo, err = collection.ByName(name)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	// including snapshot resource key
	resources := []string{string(repo.Key()), "S" + b.Name}
	taskName := fmt.Sprintf("Create snapshot of mirror %s", name)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, detail *task.Detail) (int, error) {
		err := repo.CheckLock()
		if err != nil {
			return http.StatusConflict, err
		}

		err = collection.LoadComplete(repo)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		snapshot, err = deb.NewSnapshotFromRepository(b.Name, repo)
		if err != nil {
			return http.StatusBadRequest, err
		}

		if b.Description != "" {
			snapshot.Description = b.Description
		}

		err = snapshotCollection.Add(snapshot)
		if err != nil {
			return http.StatusBadRequest, err
		}
		detail.Store(snapshot)
		return http.StatusCreated, nil
	})
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
	var resources []string

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

		resources = append(resources, string(sources[i].ResourceKey()))
	}

	maybeRunTaskInBackground(c, "Create snapshot "+b.Name, resources, func(out aptly.Progress, detail *task.Detail) (int, error) {
		list := deb.NewPackageList()

		// verify package refs and build package list
		for _, ref := range b.PackageRefs {
			p, err := collectionFactory.PackageCollection().ByKey([]byte(ref))
			if err != nil {
				if err == database.ErrNotFound {
					return http.StatusNotFound, fmt.Errorf("package %s: %s", ref, err)
				}
				return http.StatusInternalServerError, err
			}
			err = list.Add(p)
			if err != nil {
				return http.StatusBadRequest, err
			}
		}

		snapshot = deb.NewSnapshotFromRefList(b.Name, sources, deb.NewPackageRefListFromPackageList(list), b.Description)

		err = snapshotCollection.Add(snapshot)
		if err != nil {
			return http.StatusBadRequest, err
		}
		return http.StatusCreated, nil
	})
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
	name := c.Params.ByName("name")

	repo, err = collection.ByName(name)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	// including snapshot resource key
	resources := []string{string(repo.Key()), "S" + b.Name}
	taskName := fmt.Sprintf("Create snapshot of repo %s", name)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, detail *task.Detail) (int, error) {
		err := collection.LoadComplete(repo)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		snapshot, err = deb.NewSnapshotFromLocalRepo(b.Name, repo)
		if err != nil {
			return http.StatusNotFound, err
		}

		if b.Description != "" {
			snapshot.Description = b.Description
		}

		err = snapshotCollection.Add(snapshot)
		if err != nil {
			return http.StatusBadRequest, err
		}
		detail.Store(snapshot)
		return http.StatusCreated, nil
	})
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
	name := c.Params.ByName("name")

	snapshot, err = collection.ByName(name)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	resources := []string{string(snapshot.ResourceKey()), "S" + b.Name}
	taskName := fmt.Sprintf("Update snapshot %s", name)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, detail *task.Detail) (int, error) {
		_, err := collection.ByName(b.Name)
		if err == nil {
			return http.StatusConflict, fmt.Errorf("unable to rename: snapshot %s already exists", b.Name)
		}

		if b.Name != "" {
			snapshot.Name = b.Name
		}

		if b.Description != "" {
			snapshot.Description = b.Description
		}

		err = collectionFactory.SnapshotCollection().Update(snapshot)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		detail.Store(snapshot)
		return http.StatusOK, nil
	})
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

	resources := []string{string(snapshot.ResourceKey())}
	taskName := fmt.Sprintf("Delete snapshot %s", name)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, detail *task.Detail) (int, error) {
		published := publishedCollection.BySnapshot(snapshot)

		if len(published) > 0 {
			return http.StatusConflict, fmt.Errorf("unable to drop: snapshot is published")
		}

		if !force {
			snapshots := snapshotCollection.BySnapshotSource(snapshot)
			if len(snapshots) > 0 {
				return http.StatusConflict, fmt.Errorf("won't delete snapshot that was used as source for other snapshots, use ?force=1 to override")
			}
		}

		err = snapshotCollection.Drop(snapshot)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		detail.Store(gin.H{})
		return http.StatusOK, nil
	})
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
