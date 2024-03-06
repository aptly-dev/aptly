package api

import (
	"fmt"
	"net/http"
	"strings"

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
		AbortWithJSONError(c, 404, err)
		return
	}

	// including snapshot resource key
	resources := []string{string(repo.Key()), "S" + b.Name}
	taskName := fmt.Sprintf("Create snapshot of mirror %s", name)
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err := repo.CheckLock()
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusConflict, Value: nil}, err
		}

		err = collection.LoadComplete(repo)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}

		snapshot, err = deb.NewSnapshotFromRepository(b.Name, repo)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusBadRequest, Value: nil}, err
		}

		if b.Description != "" {
			snapshot.Description = b.Description
		}

		err = snapshotCollection.Add(snapshot)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusBadRequest, Value: nil}, err
		}
		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: snapshot}, nil
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
			AbortWithJSONError(c, 404, err)
			return
		}

		err = snapshotCollection.LoadComplete(sources[i])
		if err != nil {
			AbortWithJSONError(c, 500, err)
			return
		}

		resources = append(resources, string(sources[i].ResourceKey()))
	}

	maybeRunTaskInBackground(c, "Create snapshot "+b.Name, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		list := deb.NewPackageList()

		// verify package refs and build package list
		for _, ref := range b.PackageRefs {
			p, err := collectionFactory.PackageCollection().ByKey([]byte(ref))
			if err != nil {
				if err == database.ErrNotFound {
					return &task.ProcessReturnValue{Code: http.StatusNotFound, Value: nil}, fmt.Errorf("package %s: %s", ref, err)
				}
				return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
			}
			err = list.Add(p)
			if err != nil {
				return &task.ProcessReturnValue{Code: http.StatusBadRequest, Value: nil}, err
			}
		}

		snapshot = deb.NewSnapshotFromRefList(b.Name, sources, deb.NewPackageRefListFromPackageList(list), b.Description)

		err = snapshotCollection.Add(snapshot)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusBadRequest, Value: nil}, err
		}
		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: snapshot}, nil
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
		AbortWithJSONError(c, 404, err)
		return
	}

	// including snapshot resource key
	resources := []string{string(repo.Key()), "S" + b.Name}
	taskName := fmt.Sprintf("Create snapshot of repo %s", name)
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err := collection.LoadComplete(repo)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}

		snapshot, err = deb.NewSnapshotFromLocalRepo(b.Name, repo)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusNotFound, Value: nil}, err
		}

		if b.Description != "" {
			snapshot.Description = b.Description
		}

		err = snapshotCollection.Add(snapshot)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusBadRequest, Value: nil}, err
		}
		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: snapshot}, nil
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
		AbortWithJSONError(c, 404, err)
		return
	}

	resources := []string{string(snapshot.ResourceKey()), "S" + b.Name}
	taskName := fmt.Sprintf("Update snapshot %s", name)
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		_, err := collection.ByName(b.Name)
		if err == nil {
			return &task.ProcessReturnValue{Code: http.StatusConflict, Value: nil}, fmt.Errorf("unable to rename: snapshot %s already exists", b.Name)
		}

		if b.Name != "" {
			snapshot.Name = b.Name
		}

		if b.Description != "" {
			snapshot.Description = b.Description
		}

		err = collectionFactory.SnapshotCollection().Update(snapshot)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}
		return &task.ProcessReturnValue{Code: http.StatusOK, Value: snapshot}, nil
	})
}

// GET /api/snapshots/:name
func apiSnapshotsShow(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()

	snapshot, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	err = collection.LoadComplete(snapshot)
	if err != nil {
		AbortWithJSONError(c, 500, err)
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
		AbortWithJSONError(c, 404, err)
		return
	}

	resources := []string{string(snapshot.ResourceKey())}
	taskName := fmt.Sprintf("Delete snapshot %s", name)
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		published := publishedCollection.BySnapshot(snapshot)

		if len(published) > 0 {
			return &task.ProcessReturnValue{Code: http.StatusConflict, Value: nil}, fmt.Errorf("unable to drop: snapshot is published")
		}

		if !force {
			snapshots := snapshotCollection.BySnapshotSource(snapshot)
			if len(snapshots) > 0 {
				return &task.ProcessReturnValue{Code: http.StatusConflict, Value: nil}, fmt.Errorf("won't delete snapshot that was used as source for other snapshots, use ?force=1 to override")
			}
		}

		err = snapshotCollection.Drop(snapshot)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}
		return &task.ProcessReturnValue{Code: http.StatusOK, Value: gin.H{}}, nil
	})
}

// GET /api/snapshots/:name/diff/:withSnapshot
func apiSnapshotsDiff(c *gin.Context) {
	onlyMatching := c.Request.URL.Query().Get("onlyMatching") == "1"

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()

	snapshotA, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	snapshotB, err := collection.ByName(c.Params.ByName("withSnapshot"))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	err = collection.LoadComplete(snapshotA)
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	err = collection.LoadComplete(snapshotB)
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	// Calculate diff
	diff, err := snapshotA.RefList().Diff(snapshotB.RefList(), collectionFactory.PackageCollection())
	if err != nil {
		AbortWithJSONError(c, 500, err)
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
		AbortWithJSONError(c, 404, err)
		return
	}

	err = collection.LoadComplete(snapshot)
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	showPackages(c, snapshot.RefList(), collectionFactory)
}

// POST /api/snapshots/merge
func apiSnapshotsMerge(c *gin.Context) {
	var (
		err      error
		snapshot *deb.Snapshot
	)

	var body struct {
		Destination string   `binding:"required"`
		Sources     []string `binding:"required"`
	}

	if c.Bind(&body) != nil {
		return
	}

	if len(body.Sources) < 1 {
		AbortWithJSONError(c, http.StatusBadRequest, fmt.Errorf("At least one source snapshot is required"))
		return
	}

	latest := c.Request.URL.Query().Get("latest") == "1"
	noRemove := c.Request.URL.Query().Get("no-remove") == "1"
	overrideMatching := !latest && !noRemove

	if noRemove && latest {
		AbortWithJSONError(c, http.StatusBadRequest, fmt.Errorf("no-remove and latest are mutually exclusive"))
		return
	}

	collectionFactory := context.NewCollectionFactory()
	snapshotCollection := collectionFactory.SnapshotCollection()

	sources := make([]*deb.Snapshot, len(body.Sources))
	resources := make([]string, len(sources))
	for i := range body.Sources {
		sources[i], err = snapshotCollection.ByName(body.Sources[i])
		if err != nil {
			AbortWithJSONError(c, http.StatusNotFound, err)
			return
		}

		err = snapshotCollection.LoadComplete(sources[i])
		if err != nil {
			AbortWithJSONError(c, http.StatusInternalServerError, err)
			return
		}
		resources[i] = string(sources[i].ResourceKey())
	}

	maybeRunTaskInBackground(c, "Merge snapshot "+body.Destination, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		result := sources[0].RefList()
		for i := 1; i < len(sources); i++ {
			result = result.Merge(sources[i].RefList(), overrideMatching, false)
		}

		if latest {
			result.FilterLatestRefs()
		}

		sourceDescription := make([]string, len(sources))
		for i, s := range sources {
			sourceDescription[i] = fmt.Sprintf("'%s'", s.Name)
		}

		snapshot = deb.NewSnapshotFromRefList(body.Destination, sources, result,
			fmt.Sprintf("Merged from sources: %s", strings.Join(sourceDescription, ", ")))

		err = collectionFactory.SnapshotCollection().Add(snapshot)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to create snapshot: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: snapshot}, nil
	})
}
