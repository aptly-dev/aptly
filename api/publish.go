package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/task"
	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
)

type signingParams struct {
	// Don't sign published repository
	Skip bool `                json:"Skip"`
	// GPG key ID to use when signing the release, if not specified default key is used
	GpgKey string `            json:"GpgKey"`
	// GPG keyring to use (instead of default)
	Keyring string `           json:"Keyring"`
	// GPG secret keyring to use (instead of default)
	SecretKeyring string `    json:"SecretKeyring"`
	// GPG passphrase to unlock private key (possibly insecure)
	Passphrase string `       json:"Passphrase"`
	// GPG passphrase file to unlock private key (possibly insecure)
	PassphraseFile string `    json:"PassphraseFile"`
}

type sourceParams struct {
	// Name of the component
	Component string `binding:"required"   json:"Component"`
	// Name of the local repository/snapshot
	Name string `binding:"required"        json:"Name"`
}

func getSigner(options *signingParams) (pgp.Signer, error) {
	if options.Skip {
		return nil, nil
	}

	signer := context.GetSigner()
	signer.SetKey(options.GpgKey)
	signer.SetKeyRing(options.Keyring, options.SecretKeyring)
	signer.SetPassphrase(options.Passphrase, options.PassphraseFile)

	// If Batch is false, GPG will ask for passphrase on stdin, which would block the api process
	signer.SetBatch(true)

	err := signer.Init()
	if err != nil {
		return nil, err
	}

	return signer, nil
}

// Replace '_' with '/' and double '__' with single '_', SanitizePath
func slashEscape(path string) string {
	result := strings.Replace(strings.Replace(path, "_", "/", -1), "//", "_", -1)
	result = utils.SanitizePath(result)
	if result == "" {
		result = "."
	}
	return result
}

// @Summary List published repositories
// @Description **List published repositories**
// @Description
// @Description Lists repositories that have been published based on local repositories or snapshots. For each repository information about `endpoint`, `prefix` and `distribution` is listed along with `component` and architecture list. Information about snapshot or local repo being published is appended to published repository description.
// @Tags Publish
// @Produce json
// @Success 200 {array} deb.PublishedRepo
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish [get]
func apiPublishList(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	repos := make([]*deb.PublishedRepo, 0, collection.Len())

	err := collection.ForEach(func(repo *deb.PublishedRepo) error {
		err := collection.LoadShallow(repo, collectionFactory)
		if err != nil {
			return err
		}

		repos = append(repos, repo)

		return nil
	})

	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, repos)
}

// @Summary Show published repository
// @Description **Show published repository**
// @Description
// @Description Show detailed information of published repository.
// @Tags Publish
// @Produce json
// @Param prefix path string true "publishing prefix, use `:.` instead of `.` because it is ambigious in URLs"
// @Param distribution path string true "distribution name"
// @Success 200 {object} deb.PublishedRepo
// @Failure 404 {object} Error "Published repository not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution} [get]
func apiPublishShow(c *gin.Context) {
	param := slashEscape(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := slashEscape(c.Params.ByName("distribution"))

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to show: %s", err))
		return
	}

	err = collection.LoadComplete(published, collectionFactory)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unable to show: %s", err))
		return
	}

	c.JSON(http.StatusOK, published)
}

type publishedRepoCreateParams struct {
	// 'local' for local repositories and 'snapshot' for snapshots
	SourceKind string `binding:"required"    json:"SourceKind"    example:"snapshot"`
	// List of 'Component/Name' objects, 'Name' is either local repository or snapshot name
	Sources []sourceParams `binding:"required"    json:"Sources"`
	// Distribution name, if missing Aptly would try to guess from sources
	Distribution string `                         json:"Distribution"`
	// Value of Label: field in published repository stanza
	Label string `                                json:"Label"`
	// Value of Origin: field in published repository stanza
	Origin string `                               json:"Origin"`
	// when publishing, overwrite files in pool/ directory without notice
	ForceOverwrite bool `                         json:"ForceOverwrite"`
	// Override list of published architectures
	Architectures []string `                      json:"Architectures"`
	// GPG options
	Signing signingParams `                       json:"Signing"`
	// Setting to yes indicates to the package manager to not install or upgrade packages from the repository without user consent
	NotAutomatic string `                         json:"NotAutomatic"`
	// setting to yes excludes upgrades from the NotAutomic setting
	ButAutomaticUpgrades string `                 json:"ButAutomaticUpgrades"`
	// Don't generate contents indexes
	SkipContents *bool `                          json:"SkipContents"`
	// Don't remove unreferenced files in prefix/component
	SkipCleanup *bool `                           json:"SkipCleanup"`
	// Skip bz2 compression for index files
	SkipBz2 *bool `                               json:"SkipBz2"`
	// Provide index files by hash
	AcquireByHash *bool `                         json:"AcquireByHash"`
	// Enable multiple packages with the same filename in different distributions
	MultiDist *bool `                             json:"MultiDist"`
}

// @Summary Create published repository
// @Description **Publish local repository or snapshot under specified prefix**
// @Description
// @Description Storage might be passed in prefix as well, e.g. `s3:packages/`. To supply empty prefix, just remove last part (`POST /api/publish`).
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Consume json
// @Param request body publishedRepoCreateParams true "Parameters"
// @Produce json
// @Success 200 {object} deb.PublishedRepo
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Source not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix} [post]
func apiPublishRepoOrSnapshot(c *gin.Context) {
	var (
		b          publishedRepoCreateParams
		components []string
		names      []string
		sources    []interface{}
		resources  []string
	)

	param := slashEscape(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)

	if c.Bind(&b) != nil {
		return
	}

	b.Distribution = utils.SanitizePath(b.Distribution)

	var archs []string
	for _, arch := range b.Architectures {
		archs = append(archs, utils.SanitizePath(arch))
	}
	b.Architectures = archs

	signer, err := getSigner(&b.Signing)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unable to initialize GPG signer: %s", err))
		return
	}

	if len(b.Sources) == 0 {
		AbortWithJSONError(c, http.StatusBadRequest, fmt.Errorf("unable to publish: sources are empty"))
		return
	}

	collectionFactory := context.NewCollectionFactory()

	if b.SourceKind == deb.SourceSnapshot {
		var snapshot *deb.Snapshot

		snapshotCollection := collectionFactory.SnapshotCollection()

		for _, source := range b.Sources {
			components = append(components, source.Component)
			names = append(names, source.Name)

			snapshot, err = snapshotCollection.ByName(source.Name)
			if err != nil {
				AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to publish: %s", err))
				return
			}

			resources = append(resources, string(snapshot.ResourceKey()))
			sources = append(sources, snapshot)
		}
	} else if b.SourceKind == deb.SourceLocalRepo {
		var localRepo *deb.LocalRepo

		localCollection := collectionFactory.LocalRepoCollection()

		for _, source := range b.Sources {
			components = append(components, source.Component)
			names = append(names, source.Name)

			localRepo, err = localCollection.ByName(source.Name)
			if err != nil {
				AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to publish: %s", err))
				return
			}

			resources = append(resources, string(localRepo.Key()))
			sources = append(sources, localRepo)
		}
	} else {
		AbortWithJSONError(c, http.StatusBadRequest, fmt.Errorf("unknown SourceKind"))
		return
	}

	multiDist := false
	if b.MultiDist != nil {
		multiDist = *b.MultiDist
	}

	collection := collectionFactory.PublishedRepoCollection()

	resources = append(resources, string(published.Key()))
	taskName := fmt.Sprintf("Publish %s repository %s/%s with components \"%s\" and sources \"%s\"",
		b.SourceKind, published.StoragePrefix(), published.Distribution, strings.Join(components, `", "`), strings.Join(names, `", "`))
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		taskDetail := task.PublishDetail{
			Detail: detail,
		}
		publishOutput := &task.PublishOutput{
			Progress:      out,
			PublishDetail: taskDetail,
		}

		for _, source := range sources {
			switch s := source.(type) {
			case *deb.Snapshot:
				snapshotCollection := collectionFactory.SnapshotCollection()
				err = snapshotCollection.LoadComplete(s)
			case *deb.LocalRepo:
				localCollection := collectionFactory.LocalRepoCollection()
				err = localCollection.LoadComplete(s)
			default:
				err = fmt.Errorf("unexpected type for source: %T", source)
			}
			if err != nil {
				return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to publish: %s", err)
			}
		}

		published, err := deb.NewPublishedRepo(storage, prefix, b.Distribution, b.Architectures, components, sources, collectionFactory, multiDist)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to publish: %s", err)
		}

		resources = append(resources, string(published.Key()))

		if b.Origin != "" {
			published.Origin = b.Origin
		}
		if b.NotAutomatic != "" {
			published.NotAutomatic = b.NotAutomatic
		}
		if b.ButAutomaticUpgrades != "" {
			published.ButAutomaticUpgrades = b.ButAutomaticUpgrades
		}
		published.Label = b.Label

		published.SkipContents = context.Config().SkipContentsPublishing
		if b.SkipContents != nil {
			published.SkipContents = *b.SkipContents
		}

		published.SkipBz2 = context.Config().SkipBz2Publishing
		if b.SkipBz2 != nil {
			published.SkipBz2 = *b.SkipBz2
		}

		if b.AcquireByHash != nil {
			published.AcquireByHash = *b.AcquireByHash
		}

		duplicate := collection.CheckDuplicate(published)
		if duplicate != nil {
			collectionFactory.PublishedRepoCollection().LoadComplete(duplicate, collectionFactory)
			return &task.ProcessReturnValue{Code: http.StatusBadRequest, Value: nil}, fmt.Errorf("prefix/distribution already used by another published repo: %s", duplicate)
		}

		err = published.Publish(context.PackagePool(), context, collectionFactory, signer, publishOutput, b.ForceOverwrite)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to publish: %s", err)
		}

		err = collection.Add(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: published}, nil
	})
}

type publishedRepoUpdateSwitchParams struct {
	// when publishing, overwrite files in pool/ directory without notice
	ForceOverwrite bool `                         json:"ForceOverwrite"`
	// GPG options
	Signing signingParams `                       json:"Signing"`
	// Don't generate contents indexes
	SkipContents *bool `                          json:"SkipContents"`
	// Skip bz2 compression for index files
	SkipBz2 *bool `                               json:"SkipBz2"`
	// Don't remove unreferenced files in prefix/component
	SkipCleanup *bool `                           json:"SkipCleanup"`
	// only when updating published snapshots, list of objects 'Component/Name'
	Snapshots []sourceParams `                    json:"Snapshots"`
	// Provide index files by hash
	AcquireByHash *bool `                         json:"AcquireByHash"`
	// Enable multiple packages with the same filename in different distributions
	MultiDist *bool `                             json:"MultiDist"`
}

// @Summary Update published repository
// @Description **Update a published local repository or switch published snapshot**
// @Description
// @Description API action depends on published repository contents:
// @Description * if local repository has been published, published repository would be updated to match local repository contents
// @Description * if snapshots have been been published, it is possible to switch each component to new snapshot
// @Tags Publish
// @Produce json
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Consume json
// @Param request body publishedRepoUpdateSwitchParams true "Parameters"
// @Produce json
// @Success 200 {object} deb.PublishedRepo
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository or source not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution} [put]
func apiPublishUpdateSwitch(c *gin.Context) {
	var b publishedRepoUpdateSwitchParams

	param := slashEscape(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := slashEscape(c.Params.ByName("distribution"))

	if c.Bind(&b) != nil {
		return
	}

	signer, err := getSigner(&b.Signing)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unable to initialize GPG signer: %s", err))
		return
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()
	snapshotCollection := collectionFactory.SnapshotCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to update: %s", err))
		return
	}

	var updatedComponents []string
	var updatedSnapshots []string

	if published.SourceKind == deb.SourceLocalRepo {
		if len(b.Snapshots) > 0 {
			AbortWithJSONError(c, http.StatusBadRequest, fmt.Errorf("snapshots shouldn't be given when updating local repo"))
			return
		}
		updatedComponents = published.Components()
	} else if published.SourceKind == "snapshot" {
		for _, snapshotInfo := range b.Snapshots {
			snapshot, err2 := snapshotCollection.ByName(snapshotInfo.Name)
			if err2 != nil {
				AbortWithJSONError(c, http.StatusNotFound, err2)
				return
			}
			updatedComponents = append(updatedComponents, snapshotInfo.Component)
			updatedSnapshots = append(updatedSnapshots, snapshot.Name)
		}
	} else {
		AbortWithJSONError(c, 500, fmt.Errorf("unknown published repository type"))
		return
	}

	if b.SkipContents != nil {
		published.SkipContents = *b.SkipContents
	}

	if b.SkipBz2 != nil {
		published.SkipBz2 = *b.SkipBz2
	}

	if b.AcquireByHash != nil {
		published.AcquireByHash = *b.AcquireByHash
	}

	if b.MultiDist != nil {
		published.MultiDist = *b.MultiDist
	}

	revision := published.ObtainRevision()
	sources := revision.Sources

	if published.SourceKind == deb.SourceSnapshot {
		for _, snapshotInfo := range b.Snapshots {
			component := snapshotInfo.Component
			name := snapshotInfo.Name
			sources[component] = name
		}
	}

	resources := []string{string(published.Key())}
	taskName := fmt.Sprintf("Update published %s repository %s/%s", published.SourceKind, published.StoragePrefix(), published.Distribution)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collection.LoadComplete(published, collectionFactory)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("Unable to update: %s", err)
		}

		if published.SourceKind == deb.SourceLocalRepo {
			for _, component := range updatedComponents {
				published.UpdateLocalRepo(component)
			}
		} else if published.SourceKind == "snapshot" {
			for _, snapshotInfo := range b.Snapshots {
				snapshot, err2 := snapshotCollection.ByName(snapshotInfo.Name)
				if err2 != nil {
					return &task.ProcessReturnValue{Code: http.StatusNotFound, Value: nil}, err2
				}

				err2 = snapshotCollection.LoadComplete(snapshot)
				if err2 != nil {
					return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, err2
				}

				published.UpdateSnapshot(snapshotInfo.Component, snapshot)
			}
		}

		result, err := published.Update(collectionFactory, out)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("Unable to update: %s", err)
		}

		err = published.Publish(context.PackagePool(), context, collectionFactory, signer, out, b.ForceOverwrite)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("Unable to update: %s", err)
		}

		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		if b.SkipCleanup == nil || !*b.SkipCleanup {
			cleanComponents := make([]string, 0, len(result.UpdatedSources)+len(result.RemovedSources))
			cleanComponents = append(append(cleanComponents, result.UpdatedComponents()...), result.RemovedComponents()...)
			err = collection.CleanupPrefixComponentFiles(context, published, cleanComponents, collectionFactory, out)
			if err != nil {
				return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
			}
		}

		return &task.ProcessReturnValue{Code: http.StatusOK, Value: published}, nil
	})
}

// @Summary Delete published repository
// @Description **Delete a published repository**
// @Description
// @Description Delete published repository and clean up files in published directory. Aptly tries to remove as many files belonging to this repository as possible. For example, if no other published repositories share the same prefix, all files inside the prefix will be removed.
// @Tags Publish
// @Produce json
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Param force query int true "force: 1 to enable"
// @Param skipCleanup query int true "skipCleanup: 1 to enable"
// @Success 200
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution} [delete]
func apiPublishDrop(c *gin.Context) {
	param := slashEscape(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := slashEscape(c.Params.ByName("distribution"))

	force := c.Request.URL.Query().Get("force") == "1"
	skipCleanup := c.Request.URL.Query().Get("SkipCleanup") == "1"

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to drop: %s", err))
		return
	}

	resources := []string{string(published.Key())}
	taskName := fmt.Sprintf("Delete published %s repository %s/%s", published.SourceKind, published.StoragePrefix(), published.Distribution)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err := collection.Remove(context, storage, prefix, distribution,
			collectionFactory, out, force, skipCleanup)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to drop: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusOK, Value: gin.H{}}, nil
	})
}

// @Summary Add source to staged source list
// @Description **Add a source to the staged source list**
// @Description
// @Description The staged source list exists independently of the current source list of the published repository. It can be modified in multiple steps by adding, removing and updating sources. A source is a tuple of two elements comprising the name of the component and the name of the local repository or snapshot. The staged source list exists as long as it gets discarded via `drop` or applied to the published repository via `update`.
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Consume json
// @Param request body sourceParams true "Parameters"
// @Produce json
// @Success 200 {object} sourceParams
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources [post]
func apiPublishSourcesCreate(c *gin.Context) {
	var b sourceParams

	param := slashEscape(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := slashEscape(c.Params.ByName("distribution"))

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to create: %s", err))
		return
	}

	err = collection.LoadComplete(published, collectionFactory)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unable to create: %s", err))
		return
	}

	if c.Bind(&b) != nil {
		return
	}

	revision := published.ObtainRevision()
	sources := revision.Sources

	component := b.Component
	name := b.Name

	_, exists := sources[component]
	if exists {
		AbortWithJSONError(c, http.StatusBadRequest, fmt.Errorf("unable to create: Component '%s' already exists", component))
		return
	}

	sources[component] = name

	resources := []string{string(published.Key())}
	taskName := fmt.Sprintf("Update published %s repository %s/%s", published.SourceKind, published.StoragePrefix(), published.Distribution)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: published}, nil
	})
}

// @Summary Get staged source list
// @Description **Get the staged source list**
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Produce json
// @Success 200 {array} sourceParams
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository not found or staged source list does not exist"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources [get]
func apiPublishSourcesList(c *gin.Context) {
	param := slashEscape(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := slashEscape(c.Params.ByName("distribution"))

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to show: %s", err))
		return
	}

	err = collection.LoadComplete(published, collectionFactory)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unable to show: %s", err))
		return
	}

	revision := published.Revision
	if revision == nil {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to show: No source changes exist"))
		return
	}

	c.JSON(http.StatusOK, revision.SourceList())
}

// @Summary Set staged source list
// @Description **Set the staged source list**
// @Description
// @Description If the staged source list is known in advance, it can set via this method in a single call. All modifications done before are lost and the staged source list get replaced by the one given in the request body.
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Consume json
// @Param request body []sourceParams true "Parameters"
// @Produce json
// @Success 200 {array} sourceParams
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources [put]
func apiPublishSourcesUpdate(c *gin.Context) {
	var b []sourceParams

	param := slashEscape(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := slashEscape(c.Params.ByName("distribution"))

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to update: %s", err))
		return
	}

	err = collection.LoadComplete(published, collectionFactory)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unable to update: %s", err))
		return
	}

	if c.Bind(&b) != nil {
		return
	}

	revision := published.ObtainRevision()
	sources := make(map[string]string, len(b))
	revision.Sources = sources

	for _, source := range b {
		component := source.Component
		name := source.Name
		sources[component] = name
	}

	resources := []string{string(published.Key())}
	taskName := fmt.Sprintf("Update published %s repository %s/%s", published.SourceKind, published.StoragePrefix(), published.Distribution)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusOK, Value: revision.SourceList()}, nil
	})
}

// @Summary Delete staged source list
// @Description **Delete the staged source list**
// @Description
// @Description Delete/Discard the staged sources and keep existing sources of published repository.
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Produce json
// @Success 200
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources [delete]
func apiPublishSourcesDelete(c *gin.Context) {
	param := slashEscape(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := slashEscape(c.Params.ByName("distribution"))

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to delete: %s", err))
		return
	}

	err = collection.LoadComplete(published, collectionFactory)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unable to delete: %s", err))
		return
	}

	published.DropRevision()

	resources := []string{string(published.Key())}
	taskName := fmt.Sprintf("Update published %s repository %s/%s", published.SourceKind, published.StoragePrefix(), published.Distribution)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusOK, Value: gin.H{}}, nil
	})
}

// @Summary Update staged source
// @Description **Update the staged source of a component**
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Param component path string true "component name"
// @Consume json
// @Param request body sourceParams true "Parameters"
// @Produce json
// @Success 200
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository/component not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources/{component} [put]
func apiPublishSourceUpdate(c *gin.Context) {
	var b sourceParams

	param := slashEscape(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := slashEscape(c.Params.ByName("distribution"))
	component := slashEscape(c.Params.ByName("component"))

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to update: %s", err))
		return
	}

	err = collection.LoadComplete(published, collectionFactory)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unable to update: %s", err))
		return
	}

	revision := published.ObtainRevision()
	sources := revision.Sources

	_, exists := sources[component]
	if !exists {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to update: Component '%s' does not exist", component))
		return
	}

	b.Component = component
	b.Name = revision.Sources[component]

	if c.Bind(&b) != nil {
		return
	}

	if b.Component != component {
		delete(sources, component)
	}

	component = b.Component
	name := b.Name
	sources[component] = name

	resources := []string{string(published.Key())}
	taskName := fmt.Sprintf("Update published %s repository %s/%s", published.SourceKind, published.StoragePrefix(), published.Distribution)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusOK, Value: gin.H{}}, nil
	})
}

// @Summary Delete source from staged source list
// @Description **Delete a single source from the staged source list**
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Param component path string true "component name"
// @Produce json
// @Success 200
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources/{component} [delete]
func apiPublishSourceDelete(c *gin.Context) {
	param := slashEscape(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := slashEscape(c.Params.ByName("distribution"))
	component := slashEscape(c.Params.ByName("component"))

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to delete: %s", err))
		return
	}

	err = collection.LoadComplete(published, collectionFactory)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unable to delete: %s", err))
		return
	}

	revision := published.ObtainRevision()
	sources := revision.Sources

	_, exists := sources[component]
	if !exists {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to delete: Component '%s' does not exist", component))
		return
	}

	delete(sources, component)

	resources := []string{string(published.Key())}
	taskName := fmt.Sprintf("Update published %s repository %s/%s", published.SourceKind, published.StoragePrefix(), published.Distribution)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusOK, Value: gin.H{}}, nil
	})
}

type publishedRepoUpdateParams struct {
	// when publishing, overwrite files in pool/ directory without notice
	ForceOverwrite bool `                         json:"ForceOverwrite"`
	// GPG options
	Signing signingParams `                       json:"Signing"`
	// Don't generate contents indexes
	SkipContents *bool `                          json:"SkipContents"`
	// Skip bz2 compression for index files
	SkipBz2 *bool `                               json:"SkipBz2"`
	// Don't remove unreferenced files in prefix/component
	SkipCleanup *bool `                           json:"SkipCleanup"`
	// Provide index files by hash
	AcquireByHash *bool `                         json:"AcquireByHash"`
	// Enable multiple packages with the same filename in different distributions
	MultiDist *bool `                             json:"MultiDist"`
}

// @Summary Update content of published repository
// @Description **Update the content of a published repository**
// @Description
// @Description Replace the current source list of the published repository by the staged one (if available) and (re-)publish the new content.
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Consume json
// @Param request body publishedRepoUpdateParams true "Parameters"
// @Produce json
// @Success 200 {object} deb.PublishedRepo
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository/component not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/update [post]
func apiPublishUpdate(c *gin.Context) {
	var b publishedRepoUpdateParams

	param := slashEscape(c.Params.ByName("prefix"))
	storage, prefix := deb.ParsePrefix(param)
	distribution := slashEscape(c.Params.ByName("distribution"))

	if c.Bind(&b) != nil {
		return
	}

	signer, err := getSigner(&b.Signing)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unable to initialize GPG signer: %s", err))
		return
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PublishedRepoCollection()

	published, err := collection.ByStoragePrefixDistribution(storage, prefix, distribution)
	if err != nil {
		AbortWithJSONError(c, http.StatusNotFound, fmt.Errorf("unable to update: %s", err))
		return
	}

	err = collection.LoadComplete(published, collectionFactory)
	if err != nil {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unable to update: %s", err))
		return
	}

	if b.SkipContents != nil {
		published.SkipContents = *b.SkipContents
	}

	if b.SkipBz2 != nil {
		published.SkipBz2 = *b.SkipBz2
	}

	if b.AcquireByHash != nil {
		published.AcquireByHash = *b.AcquireByHash
	}

	if b.MultiDist != nil {
		published.MultiDist = *b.MultiDist
	}

	resources := []string{string(published.Key())}
	taskName := fmt.Sprintf("Update published %s repository %s/%s", published.SourceKind, published.StoragePrefix(), published.Distribution)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		result, err := published.Update(collectionFactory, out)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
		}

		err = published.Publish(context.PackagePool(), context, collectionFactory, signer, out, b.ForceOverwrite)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
		}

		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		if b.SkipCleanup == nil || !*b.SkipCleanup {
			cleanComponents := make([]string, 0, len(result.UpdatedSources)+len(result.RemovedSources))
			cleanComponents = append(append(cleanComponents, result.UpdatedComponents()...), result.RemovedComponents()...)
			err = collection.CleanupPrefixComponentFiles(context, published, cleanComponents, collectionFactory, out)
			if err != nil {
				return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
			}
		}

		return &task.ProcessReturnValue{Code: http.StatusOK, Value: published}, nil
	})
}
