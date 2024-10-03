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

// @Summary Get snapshots
// @Description Get list of available snapshots. Each snapshot is returned as in “show” API.
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
// @Param name path string true "Name of the snapshot to be created"
// @Param latest query int false "merge only the latest version of each package"
// @Param no-remove query int false "all versions of packages are preserved during merge"
// @Consume json
// @Param request body snapshotsMergeParams true "Parameters"
// @Produce  json
// @Success 200
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

		err = snapshotCollection.LoadComplete(sources[i])
		if err != nil {
			AbortWithJSONError(c, http.StatusInternalServerError, err)
			return
		}
		resources[i] = string(sources[i].ResourceKey())
	}

	maybeRunTaskInBackground(c, "Merge snapshot "+name, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
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

		snapshot = deb.NewSnapshotFromRefList(name, sources, result,
			fmt.Sprintf("Merged from sources: %s", strings.Join(sourceDescription, ", ")))

		err = collectionFactory.SnapshotCollection().Add(snapshot)
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
// @Param name path string true "Name of the snapshot to be created"
// @Param all-matches query int false "pull all the packages that satisfy the dependency version requirements (default is to pull first matching package): 1 to enable"
// @Param dry-run query int false "don’t create destination snapshot, just show what would be pulled: 1 to enable"
// @Param no-deps query int false "don’t process dependencies, just pull listed packages: 1 to enable"
// @Param no-remove query int false "don’t remove other package versions when pulling package: 1 to enable"
// @Consume json
// @Param request body snapshotsPullParams true "Parameters"
// @Produce json
// @Success 200
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
	err = collectionFactory.SnapshotCollection().LoadComplete(toSnapshot)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, err)
		return
	}

	// Load <Source> snapshot
	sourceSnapshot, err := collectionFactory.SnapshotCollection().ByName(body.Source)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, err)
		return
	}
	err = collectionFactory.SnapshotCollection().LoadComplete(sourceSnapshot)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, err)
		return
	}

	resources := []string{string(sourceSnapshot.ResourceKey()), string(toSnapshot.ResourceKey())}
	taskName := fmt.Sprintf("Pull snapshot %s into %s and save as %s", body.Source, name, body.Destination)
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
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
		destinationPackageList, err := sourcePackageList.FilterWithProgress(queries, !noDeps, toPackageList, context.DependencyOptions(), architecturesList, context.Progress())
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
				packageSearchResults := toPackageList.Search(deb.Dependency{Architecture: pkg.Architecture, Pkg: pkg.Name}, true)
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
