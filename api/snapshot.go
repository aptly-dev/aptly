package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
	"github.com/aptly-dev/aptly/task"
	"github.com/gin-gonic/gin"
)

// @Summary List Snapshots
// @Description **Get list of snapshots**
// @Description
// @Description Each snapshot is returned as in “show” API.
// @Tags Snapshots
// @Produce  json
// @Success 200 {array} deb.Snapshot
// @Router /api/snapshots [get]
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

type snapshotsCreateFromMirrorParams struct {
	// Name of snapshot to create
	Name string `binding:"required"     json:"Name"                 example:"snap1"`
	// Description of snapshot
	Description string `                json:"Description"`
}

// @Summary Snapshot Mirror
// @Description **Create a snapshot of a mirror**
// @Tags Snapshots
// @Produce json
// @Param request body snapshotsCreateFromMirrorParams true "Parameters"
// @Param name path string true "Mirror name"
// @Param _async query bool false "Run in background and return task object"
// @Success 201 {object} deb.Snapshot "Created Snapshot"
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Mirror Not Found"
// @Failure 409 {object} Error "Conflicting snapshot"
// @Failure 500 {object} Error "Internal Server Error"
// @Router /api/mirrors/{name}/snapshots [post]
func apiSnapshotsCreateFromMirror(c *gin.Context) {
	var (
		err      error
		repo     *deb.RemoteRepo
		snapshot *deb.Snapshot
		b        snapshotsCreateFromMirrorParams
	)

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

		err = collection.LoadComplete(repo, collectionFactory.RefListCollection())
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

		err = snapshotCollection.Add(snapshot, collectionFactory.RefListCollection())
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusBadRequest, Value: nil}, err
		}
		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: snapshot}, nil
	})
}

type snapshotsCreateParams struct {
	// Name of snapshot to create
	Name string `binding:"required"  json:"Name"                 example:"snap2"`
	// Description of snapshot
	Description string `             json:"Description"`
	// List of source snapshots
	SourceSnapshots []string `       json:"SourceSnapshots"      example:"snap1"`
	// List of package refs
	PackageRefs []string `           json:"PackageRefs"          example:""`
}

// @Summary Snapshot Packages
// @Description **Create a snapshot from package refs**
// @Description
// @Description Refs can be obtained from snapshots, local repos, or mirrors
// @Tags Snapshots
// @Param request body snapshotsCreateParams true "Parameters"
// @Param _async query bool false "Run in background and return task object"
// @Produce json
// @Success 201 {object} deb.Snapshot "Created snapshot"
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Source snapshot or package refs not found"
// @Failure 500 {object} Error "Internal Server Error"
// @Router /api/snapshots [post]
func apiSnapshotsCreate(c *gin.Context) {
	var (
		err      error
		snapshot *deb.Snapshot
		b        snapshotsCreateParams
	)

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

		resources = append(resources, string(sources[i].ResourceKey()))
	}

	maybeRunTaskInBackground(c, "Create snapshot "+b.Name, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		for i := range sources {
			err = snapshotCollection.LoadComplete(sources[i], collectionFactory.RefListCollection())
			if err != nil {
				return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
			}
		}

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

		snapshot = deb.NewSnapshotFromRefList(b.Name, sources, deb.NewSplitRefListFromPackageList(list), b.Description)

		err = snapshotCollection.Add(snapshot, collectionFactory.RefListCollection())
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusBadRequest, Value: nil}, err
		}
		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: snapshot}, nil
	})
}

type snapshotsCreateFromRepositoryParams struct {
	// Name of snapshot to create
	Name string `binding:"required"               json:"Name"                 example:"snap1"`
	// Description of snapshot
	Description string `                          json:"Description"`
}

// @Summary Snapshot Repository
// @Description **Create a snapshot of a repository by name**
// @Tags Snapshots
// @Param name path string true "Repository name"
// @Consume json
// @Param request body snapshotsCreateFromRepositoryParams true "Parameters"
// @Param name path string true "Name of the snapshot"
// @Param _async query bool false "Run in background and return task object"
// @Produce json
// @Success 201 {object} deb.Snapshot "Created snapshot object"
// @Failure 400 {object} Error "Bad Request"
// @Failure 500 {object} Error "Internal Server Error"
// @Failure 404 {object} Error "Repo Not Found"
// @Router /api/repos/{name}/snapshots [post]
func apiSnapshotsCreateFromRepository(c *gin.Context) {
	var (
		err      error
		repo     *deb.LocalRepo
		snapshot *deb.Snapshot
		b        snapshotsCreateFromRepositoryParams
	)

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
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		err := collection.LoadComplete(repo, collectionFactory.RefListCollection())
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

		err = snapshotCollection.Add(snapshot, collectionFactory.RefListCollection())
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusBadRequest, Value: nil}, err
		}
		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: snapshot}, nil
	})
}

type snapshotsUpdateParams struct {
	// Change Name of snapshot
	Name string `       json:"Name"  example:"snap2"`
	// Change Description of snapshot
	Description string `json:"Description"`
}

// @Summary Update Snapshot
// @Description **Update snapshot metadata (Name, Description)**
// @Tags Snapshots
// @Param request body snapshotsUpdateParams true "Parameters"
// @Param name path string true "Snapshot name"
// @Param _async query bool false "Run in background and return task object"
// @Produce json
// @Success 200 {object} deb.Snapshot "Updated snapshot object"
// @Failure 404 {object} Error "Snapshot Not Found"
// @Failure 409 {object} Error "Conflicting snapshot"
// @Failure 500 {object} Error "Internal Server Error"
// @Router /api/snapshots/{name} [put]
func apiSnapshotsUpdate(c *gin.Context) {
	var (
		err      error
		snapshot *deb.Snapshot
		b        snapshotsUpdateParams
	)

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

		err = collectionFactory.SnapshotCollection().Update(snapshot, collectionFactory.RefListCollection())
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}
		return &task.ProcessReturnValue{Code: http.StatusOK, Value: snapshot}, nil
	})
}

// @Summary Get Snapshot Info
// @Description **Query detailed information about a snapshot by name**
// @Tags Snapshots
// @Param name path string true "Name of the snapshot"
// @Produce json
// @Success 200 {object} deb.Snapshot "msg"
// @Failure 404 {object} Error "Snapshot Not Found"
// @Failure 500 {object} Error "Internal Server Error"
// @Router /api/snapshots/{name} [get]
func apiSnapshotsShow(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()

	snapshot, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	err = collection.LoadComplete(snapshot, collectionFactory.RefListCollection())
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	c.JSON(200, snapshot)
}

// @Summary Delete Snapshot
// @Description **Delete snapshot by name**
// @Description Cannot drop snapshots that are published.
// @Description Needs force=1 to drop snapshots used as source by other snapshots.
// @Tags Snapshots
// @Param name path string true "Snapshot name"
// @Param force query string false "Force operation"
// @Param _async query bool false "Run in background and return task object"
// @Produce json
// @Success 200 ""
// @Failure 404 {object} Error "Snapshot Not Found"
// @Failure 409 {object} Error "Snapshot in use"
// @Failure 500 {object} Error "Internal Server Error"
// @Router /api/snapshots/{name} [delete]
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

// @Summary Snapshot diff
// @Description **Return the diff between two snapshots (name & withSnapshot)**
// @Description Provide `onlyMatching=1` to return only packages present in both snapshots.
// @Description Otherwise, returns a `left` and `right` result providing packages only in the first and second snapshots
// @Tags Snapshots
// @Produce json
// @Param name path string true "Snapshot name"
// @Param withSnapshot path string true "Snapshot name to diff against"
// @Param onlyMatching query string false "Only return packages present in both snapshots"
// @Success 200 {array} deb.PackageDiff "Package Diff"
// @Failure 404 {object} Error "Snapshot Not Found"
// @Failure 500 {object} Error "Internal Server Error"
// @Router /api/snapshots/{name}/diff/{withSnapshot} [get]
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

	err = collection.LoadComplete(snapshotA, collectionFactory.RefListCollection())
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	err = collection.LoadComplete(snapshotB, collectionFactory.RefListCollection())
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	// Calculate diff
	diff, err := snapshotA.RefList().Diff(snapshotB.RefList(), collectionFactory.PackageCollection(), nil)
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

// @Summary List Snapshot Packages
// @Description **List all packages in snapshot or perform search on snapshot contents and return results**
// @Description If `q` query parameter is missing, return all packages, otherwise return packages that match q
// @Tags Snapshots
// @Produce json
// @Param name path string true "Snapshot to search"
// @Param q query string false "Package query (e.g Name%20(~%20matlab))"
// @Param withDeps query string false "Set to 1 to include dependencies when evaluating package query"
// @Param format query string false "Set to 'details' to return extra info about each package"
// @Param maximumVersion query string false "Set to 1 to only return the highest version for each package name"
// @Success 200 {array} string "Package info"
// @Failure 404 {object} Error "Snapshot Not Found"
// @Failure 500 {object} Error "Internal Server Error"
// @Router /api/snapshots/{name}/packages [get]
func apiSnapshotsSearchPackages(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.SnapshotCollection()

	snapshot, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	err = collection.LoadComplete(snapshot, collectionFactory.RefListCollection())
	if err != nil {
		AbortWithJSONError(c, 500, err)
		return
	}

	showPackages(c, snapshot.RefList(), collectionFactory)
}

type snapshotsMergeParams struct {
	// List of snapshot names to be merged
	Sources []string `binding:"required" json:"Sources"     example:"snapshot1"`
}

// @Summary Snapshot Merge
// @Description **Merge several source snapshots into a new snapshot**
// @Description
// @Description Merge happens from left to right. By default, packages with the same name-architecture pair are replaced during merge (package from latest snapshot on the list wins).
// @Description
// @Description If only one snapshot is specified, merge copies source into destination.
// @Tags Snapshots
// @Consume json
// @Produce json
// @Param name path string true "Name of the snapshot to be created"
// @Param latest query int false "merge only the latest version of each package"
// @Param no-remove query int false "all versions of packages are preserved during merge"
// @Param request body snapshotsMergeParams true "Parameters"
// @Param _async query bool false "Run in background and return task object"
// @Success 201 {object} deb.Snapshot "Resulting snapshot object"
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Not Found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/snapshots/{name}/merge [post]
func apiSnapshotsMerge(c *gin.Context) {
	var (
		err      error
		snapshot *deb.Snapshot
		body     snapshotsMergeParams
	)

	name := c.Params.ByName("name")

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

		resources[i] = string(sources[i].ResourceKey())
	}

	maybeRunTaskInBackground(c, "Merge snapshot "+name, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = snapshotCollection.LoadComplete(sources[0], collectionFactory.RefListCollection())
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}
		result := sources[0].RefList()
		for i := 1; i < len(sources); i++ {
			err = snapshotCollection.LoadComplete(sources[i], collectionFactory.RefListCollection())
			if err != nil {
				return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
			}
			result = result.Merge(sources[i].RefList(), overrideMatching, false)
		}

		if latest {
			result.FilterLatestRefs()
		}

		sourceDescription := make([]string, len(sources))
		for i, s := range sources {
			sourceDescription[i] = fmt.Sprintf("'%s'", s.Name)
		}

		snapshot = deb.NewSnapshotFromRefList(name, sources, result,
			fmt.Sprintf("Merged from sources: %s", strings.Join(sourceDescription, ", ")))

		err = collectionFactory.SnapshotCollection().Add(snapshot, collectionFactory.RefListCollection())
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to create snapshot: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: snapshot}, nil
	})
}

type snapshotsPullParams struct {
	// Source name to be searched for packages and dependencies
	Source string `binding:"required"      json:"Source"            example:"source-snapshot"`
	// Name of the snapshot to be created
	Destination string `binding:"required" json:"Destination"       example:"idestination-snapshot"`
	// List of package queries (i.e. name of package to be pulled from `Source`)
	Queries []string `binding:"required"   json:"Queries"           example:"xserver-xorg"`
	// List of architectures (optional)
	Architectures []string `               json:"Architectures"     example:"amd64, armhf"`
}

// @Summary Snapshot Pull
// @Description **Pulls new packages and dependencies from a source snapshot into a new snapshot**
// @Description
// @Description May also upgrade package versions if name snapshot already contains packages being pulled. New snapshot `Destination` is created as result of this process.
// @Description If architectures are limited (with config architectures or parameter `Architectures`, only mentioned architectures are processed, otherwise aptly will process all architectures in the snapshot.
// @Description If following dependencies by source is enabled (using dependencyFollowSource config), pulling binary packages would also pull corresponding source packages as well.
// @Description By default aptly would remove packages matching name and architecture while importing: e.g. when importing software_1.3_amd64, package software_1.2.9_amd64 would be removed.
// @Description
// @Description With flag `no-remove` both package versions would stay in the snapshot.
// @Description
// @Description Aptly pulls first package matching each of package queries, but with flag -all-matches all matching packages would be pulled.
// @Tags Snapshots
// @Param request body snapshotsPullParams true "Parameters"
// @Param name path string true "Name of the snapshot to be created"
// @Param all-matches query int false "pull all the packages that satisfy the dependency version requirements (default is to pull first matching package): 1 to enable"
// @Param dry-run query int false "don’t create destination snapshot, just show what would be pulled: 1 to enable"
// @Param no-deps query int false "don’t process dependencies, just pull listed packages: 1 to enable"
// @Param no-remove query int false "don’t remove other package versions when pulling package: 1 to enable"
// @Param _async query bool false "Run in background and return task object"
// @Consume json
// @Produce json
// @Success 200 {object} deb.Snapshot "Resulting Snapshot object"
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Not Found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/snapshots/{name}/pull [post]
func apiSnapshotsPull(c *gin.Context) {
	var (
		err                 error
		destinationSnapshot *deb.Snapshot
		body                snapshotsPullParams
	)

	name := c.Params.ByName("name")

	if err = c.BindJSON(&body); err != nil {
		AbortWithJSONError(c, http.StatusBadRequest, err)
		return
	}

	allMatches := c.Request.URL.Query().Get("all-matches") == "1"
	dryRun := c.Request.URL.Query().Get("dry-run") == "1"
	noDeps := c.Request.URL.Query().Get("no-deps") == "1"
	noRemove := c.Request.URL.Query().Get("no-remove") == "1"

	collectionFactory := context.NewCollectionFactory()

	// Load <name> snapshot
	toSnapshot, err := collectionFactory.SnapshotCollection().ByName(name)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, err)
		return
	}

	// Load <Source> snapshot
	sourceSnapshot, err := collectionFactory.SnapshotCollection().ByName(body.Source)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, err)
		return
	}

	resources := []string{string(sourceSnapshot.ResourceKey()), string(toSnapshot.ResourceKey())}
	taskName := fmt.Sprintf("Pull snapshot %s into %s and save as %s", body.Source, name, body.Destination)
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collectionFactory.SnapshotCollection().LoadComplete(toSnapshot)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}
		err = collectionFactory.SnapshotCollection().LoadComplete(sourceSnapshot)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}

		// convert snapshots to package list
		toPackageList, err := deb.NewPackageListFromRefList(toSnapshot.RefList(), collectionFactory.PackageCollection(), context.Progress())
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}
		sourcePackageList, err := deb.NewPackageListFromRefList(sourceSnapshot.RefList(), collectionFactory.PackageCollection(), context.Progress())
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}

		toPackageList.PrepareIndex()
		sourcePackageList.PrepareIndex()

		var architecturesList []string

		if len(context.ArchitecturesList()) > 0 {
			architecturesList = context.ArchitecturesList()
		} else {
			architecturesList = toPackageList.Architectures(false)
		}

		architecturesList = append(architecturesList, body.Architectures...)
		sort.Strings(architecturesList)

		if len(architecturesList) == 0 {
			err := fmt.Errorf("unable to determine list of architectures, please specify explicitly")
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}

		// Build architecture query: (arch == "i386" | arch == "amd64" | ...)
		var archQuery deb.PackageQuery = &deb.FieldQuery{Field: "$Architecture", Relation: deb.VersionEqual, Value: ""}
		for _, arch := range architecturesList {
			archQuery = &deb.OrQuery{L: &deb.FieldQuery{Field: "$Architecture", Relation: deb.VersionEqual, Value: arch}, R: archQuery}
		}

		queries := make([]deb.PackageQuery, len(body.Queries))
		for i, q := range body.Queries {
			queries[i], err = query.Parse(q)
			if err != nil {
				return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
			}
			// Add architecture filter
			queries[i] = &deb.AndQuery{L: queries[i], R: archQuery}
		}

		// Filter with dependencies as requested
		destinationPackageList, err := sourcePackageList.Filter(deb.FilterOptions{
			Queries:           queries,
			WithDependencies:  !noDeps,
			Source:            toPackageList,
			DependencyOptions: context.DependencyOptions(),
			Architectures:     architecturesList,
			Progress:          context.Progress(),
		})
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}
		destinationPackageList.PrepareIndex()

		removedPackages := []string{}
		addedPackages := []string{}
		alreadySeen := map[string]bool{}

		destinationPackageList.ForEachIndexed(func(pkg *deb.Package) error {
			key := pkg.Architecture + "_" + pkg.Name
			_, seen := alreadySeen[key]

			// If we haven't seen such name-architecture pair and were instructed to remove, remove it
			if !noRemove && !seen {
				// Remove all packages with the same name and architecture
				packageSearchResults := toPackageList.Search(deb.Dependency{Architecture: pkg.Architecture, Pkg: pkg.Name}, true, false)
				for _, p := range packageSearchResults {
					toPackageList.Remove(p)
					removedPackages = append(removedPackages, p.String())
				}
			}

			// If !allMatches, add only first matching name-arch package
			if !seen || allMatches {
				toPackageList.Add(pkg)
				addedPackages = append(addedPackages, pkg.String())
			}

			alreadySeen[key] = true

			return nil
		})
		alreadySeen = nil

		if dryRun {
			response := struct {
				AddedPackages   []string `json:"added_packages"`
				RemovedPackages []string `json:"removed_packages"`
			}{
				AddedPackages:   addedPackages,
				RemovedPackages: removedPackages,
			}

			return &task.ProcessReturnValue{Code: http.StatusOK, Value: response}, nil
		}

		// Create <destination> snapshot
		destinationSnapshot = deb.NewSnapshotFromPackageList(body.Destination, []*deb.Snapshot{toSnapshot, sourceSnapshot}, toPackageList,
			fmt.Sprintf("Pulled into '%s' with '%s' as source, pull request was: '%s'", toSnapshot.Name, sourceSnapshot.Name, strings.Join(body.Queries, ", ")))

		err = collectionFactory.SnapshotCollection().Add(destinationSnapshot)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err
		}

		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: destinationSnapshot}, nil
	})
}
