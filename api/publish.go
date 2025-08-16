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
	Skip bool `                json:"Skip"           example:"false"`
	// GPG key ID(s) to use when signing the release, separated by comma, and if not specified, default configured key(s) are used
	GpgKey string `            json:"GpgKey"         example:"KEY_ID_a, KEY_ID_b"`
	// GPG keyring to use (instead of default)
	Keyring string `           json:"Keyring"        example:"trustedkeys.gpg"`
	// GPG secret keyring to use (instead of default) Note: depreciated with gpg2
	SecretKeyring string `     json:"SecretKeyring"  example:""`
	// GPG passphrase to unlock private key (possibly insecure)
	Passphrase string `        json:"Passphrase"     example:"verysecure"`
	// GPG passphrase file to unlock private key (possibly insecure)
	PassphraseFile string `    json:"PassphraseFile" example:"/etc/aptly.passphrase"`
}

type sourceParams struct {
	// Name of the component
	Component string `binding:"required"   json:"Component"  example:"main"`
	// Name of the local repository/snapshot
	Name string `binding:"required"        json:"Name"       example:"snap1"`
}

func getSigner(options *signingParams) (pgp.Signer, error) {
	if options.Skip {
		return nil, nil
	}

	signer := context.GetSigner()

	var multiGpgKeys []string
	// REST params have priority over config
	if options.GpgKey != "" {
		for _, p := range strings.Split(options.GpgKey, ",") {
			if t := strings.TrimSpace(p); t != "" {
				multiGpgKeys = append(multiGpgKeys, t)
			}
		}
	} else if len(context.Config().GpgKeys) > 0 {
		multiGpgKeys = context.Config().GpgKeys
	}
	for _, gpgKey := range multiGpgKeys {
		signer.SetKey(gpgKey)
	}
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

// @Summary List Published Repositories
// @Description **Get list of published repositories**
// @Description
// @Description Return list of published repositories including detailed information.
// @Description
// @Description See also: `aptly publish list`
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

// @Summary Show Published Repository
// @Description **Get published repository information**
// @Description
// @Description Show detailed information of a published repository.
// @Description
// @Description See also: `aptly publish show`
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
	SourceKind string `binding:"required"         json:"SourceKind"    example:"snapshot"`
	// List of 'Component/Name' objects, 'Name' is either local repository or snapshot name
	Sources []sourceParams `binding:"required"    json:"Sources"`
	// Distribution name, if missing Aptly would try to guess from sources
	Distribution string `                         json:"Distribution"          example:"bookworm"`
	// Value of Label: field in published repository stanza
	Label string `                                json:"Label"                 example:""`
	// Value of Origin: field in published repository stanza
	Origin string `                               json:"Origin"                example:""`
	// when publishing, overwrite files in pool/ directory without notice
	ForceOverwrite bool `                         json:"ForceOverwrite"        example:"false"`
	// Override list of published architectures
	Architectures []string `                      json:"Architectures"         example:"amd64,armhf"`
	// GPG options
	Signing signingParams `                       json:"Signing"`
	// Setting to yes indicates to the package manager to not install or upgrade packages from the repository without user consent
	NotAutomatic string `                         json:"NotAutomatic"          example:""`
	// setting to yes excludes upgrades from the NotAutomic setting
	ButAutomaticUpgrades string `                 json:"ButAutomaticUpgrades"  example:""`
	// Don't generate contents indexes
	SkipContents *bool `                          json:"SkipContents"          example:"false"`
	// Don't remove unreferenced files in prefix/component
	SkipCleanup *bool `                           json:"SkipCleanup"           example:"false"`
	// Skip bz2 compression for index files
	SkipBz2 *bool `                               json:"SkipBz2"               example:"false"`
	// Provide index files by hash
	AcquireByHash *bool `                         json:"AcquireByHash"         example:"false"`
	// Enable multiple packages with the same filename in different distributions
	MultiDist *bool `                             json:"MultiDist"             example:"false"`
}

// @Summary Create Published Repository
// @Description **Publish a local repository or snapshot**
// @Description
// @Description Create a published repository.
// @Description
// @Description The prefix may contain a storage specifier, e.g. `s3:packages/`, or it may also be empty to publish to the root directory.
// @Description
// @Description **Example:**
// @Description ```
// @Description $ curl -X POST -H 'Content-Type: application/json' --data '{"Distribution": "wheezy", "Sources": [{"Name": "aptly-repo"}]}' http://localhost:8080/api/publish//repos
// @Description {"Architectures":["i386"],"Distribution":"wheezy","Label":"","Origin":"","Prefix":".","SourceKind":"local","Sources":[{"Component":"main","Name":"aptly-repo"}],"Storage":""}
// @Description ```
// @Description
// @Description See also: `aptly publish create`
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param _async query bool false "Run in background and return task object"
// @Consume json
// @Param request body publishedRepoCreateParams true "Parameters"
// @Produce json
// @Success 201 {object} deb.PublishedRepo
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

	taskName := fmt.Sprintf("Publish %s repository %s/%s with components \"%s\" and sources \"%s\"",
		b.SourceKind, param, b.Distribution, strings.Join(components, `", "`), strings.Join(names, `", "`))
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
			_ = collectionFactory.PublishedRepoCollection().LoadComplete(duplicate, collectionFactory)
			return &task.ProcessReturnValue{Code: http.StatusBadRequest, Value: nil}, fmt.Errorf("prefix/distribution already used by another published repo: %s", duplicate)
		}

		err = published.Publish(context.PackagePool(), context, collectionFactory, signer, publishOutput, b.ForceOverwrite, context.SkelPath())
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
	ForceOverwrite bool `                         json:"ForceOverwrite" example:"false"`
	// GPG options
	Signing signingParams `                       json:"Signing"`
	// Don't generate contents indexes
	SkipContents *bool `                          json:"SkipContents"   example:"false"`
	// Skip bz2 compression for index files
	SkipBz2 *bool `                               json:"SkipBz2"        example:"false"`
	// Don't remove unreferenced files in prefix/component
	SkipCleanup *bool `                           json:"SkipCleanup"    example:"false"`
	// only when updating published snapshots, list of objects 'Component/Name'
	Snapshots []sourceParams `                    json:"Snapshots"`
	// Provide index files by hash
	AcquireByHash *bool `                         json:"AcquireByHash"  example:"false"`
	// Enable multiple packages with the same filename in different distributions
	MultiDist *bool `                             json:"MultiDist"      example:"false"`
}

// @Summary Update Published Repository
// @Description **Update a published repository**
// @Description
// @Description Update a published local repository or switch snapshot.
// @Description
// @Description For published local repositories:
// @Description * update to match local repository contents
// @Description
// @Description For published snapshots:
// @Description * switch components to new snapshot
// @Description
// @Description See also: `aptly publish update` / `aptly publish switch`
// @Tags Publish
// @Produce json
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Param _async query bool false "Run in background and return task object"
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

	if published.SourceKind == deb.SourceLocalRepo {
		if len(b.Snapshots) > 0 {
			AbortWithJSONError(c, http.StatusBadRequest, fmt.Errorf("snapshots shouldn't be given when updating local repo"))
			return
		}
	} else if published.SourceKind == deb.SourceSnapshot {
		for _, snapshotInfo := range b.Snapshots {
			_, err2 := snapshotCollection.ByName(snapshotInfo.Name)
			if err2 != nil {
				AbortWithJSONError(c, http.StatusNotFound, err2)
				return
			}
		}
	} else {
		AbortWithJSONError(c, http.StatusInternalServerError, fmt.Errorf("unknown published repository type"))
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
		err = collection.LoadComplete(published, collectionFactory)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
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

		result, err := published.Update(collectionFactory, out)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
		}

		err = published.Publish(context.PackagePool(), context, collectionFactory, signer, out, b.ForceOverwrite, context.SkelPath())
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

// @Summary Delete Published Repository
// @Description **Delete a published repository**
// @Description
// @Description Delete a distribution of a published repository and remove associated files.
// @Description
// @Description If no other published repositories share the same prefix, all files inside the prefix will be removed.
// @Description
// @Description See also: `aptly publish drop`
// @Tags Publish
// @Produce json
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Param force query int true "force: 1 to enable"
// @Param skipCleanup query int true "skipCleanup: 1 to enable"
// @Param _async query bool false "Run in background and return task object"
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

// @Summary Add Source Component
// @Description **Add a source component to a published repo**
// @Description
// @Description Add a component of a snapshot or local repository to be published.
// @Description
// @Description This call does not publish the changes, but rather schedules them for a subsequent publish update call (i.e `PUT /api/publish/{prefix}/{distribution}` / `POST /api/publish/{prefix}/{distribution}/update`).
// @Description
// @Description See also: `aptly publish source add`
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Param _async query bool false "Run in background and return task object"
// @Consume json
// @Param request body sourceParams true "Parameters"
// @Produce json
// @Success 201
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources [post]
func apiPublishAddSource(c *gin.Context) {
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
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusCreated, Value: gin.H{}}, nil
	})
}

// @Summary List Pending Changes
// @Description **List source component changes to be applied**
// @Description
// @Description Return added, removed or changed components of snapshots or local repository to be published.
// @Description
// @Description The changes will be applied by a subsequent publish update call (i.e. `PUT /api/publish/{prefix}/{distribution}` / `POST /api/publish/{prefix}/{distribution}/update`).
// @Description
// @Description See also: `aptly publish source list`
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Produce json
// @Success 200 {array} []deb.SourceEntry
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository pending changes not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources [get]
func apiPublishListChanges(c *gin.Context) {
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

// @Summary Replace Source Components
// @Description **Replace the source components of a published repository**
// @Description
// @Description Sets the components of snapshots or local repositories to be published. Existing Sourced will be replaced.
// @Description
// @Description This call does not publish the changes, but rather schedules them for a subsequent publish update call (i.e `PUT /api/publish/{prefix}/{distribution}` / `POST /api/publish/{prefix}/{distribution}/update`).
// @Description
// @Description See also: `aptly publish source replace`
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Param _async query bool false "Run in background and return task object"
// @Consume json
// @Param request body []sourceParams true "Parameters"
// @Produce json
// @Success 200
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources [put]
func apiPublishSetSources(c *gin.Context) {
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
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusOK, Value: revision.SourceList()}, nil
	})
}

// @Summary Discard Pending Changes
// @Description **Discard pending source component changes of a published repository**
// @Description
// @Description Remove all pending changes what would be applied with a subsequent publish update call (i.e. `PUT /api/publish/{prefix}/{distribution}` / `POST /api/publish/{prefix}/{distribution}/update`).
// @Description
// @Description See also: `aptly publish source drop`
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Param _async query bool false "Run in background and return task object"
// @Produce json
// @Success 200
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources [delete]
func apiPublishDropChanges(c *gin.Context) {
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
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusOK, Value: gin.H{}}, nil
	})
}

// @Summary Update Source Component
// @Description **Update the source component of a published repository**
// @Description
// @Description Update a component of a snapshot or local repository to be published.
// @Description
// @Description This call does not publish the changes, but rather schedules them for a subsequent publish update call (i.e `PUT /api/publish/{prefix}/{distribution}` / `POST /api/publish/{prefix}/{distribution}/update`).
// @Description
// @Description See also: `aptly publish source update`
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Param component path string true "component name"
// @Param _async query bool false "Run in background and return task object"
// @Consume json
// @Param request body sourceParams true "Parameters"
// @Produce json
// @Success 200
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository/component not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources/{component} [put]
func apiPublishUpdateSource(c *gin.Context) {
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
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusOK, Value: gin.H{}}, nil
	})
}

// @Summary Remove Source Component
// @Description **Remove a source component from a published repo**
// @Description
// @Description Remove a source component (snapshot / local repo) from a published repository.
// @Description
// @Description This call does not publish the changes, but rather schedules them for a subsequent publish update call (i.e `PUT /api/publish/{prefix}/{distribution}` / `POST /api/publish/{prefix}/{distribution}/update`).
// @Description
// @Description See also: `aptly publish source remove`
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Param component path string true "component name"
// @Param _async query bool false "Run in background and return task object"
// @Produce json
// @Success 200
// @Failure 400 {object} Error "Bad Request"
// @Failure 404 {object} Error "Published repository not found"
// @Failure 500 {object} Error "Internal Error"
// @Router /api/publish/{prefix}/{distribution}/sources/{component} [delete]
func apiPublishRemoveSource(c *gin.Context) {
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
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err = collection.Update(published)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to save to DB: %s", err)
		}

		return &task.ProcessReturnValue{Code: http.StatusOK, Value: gin.H{}}, nil
	})
}

type publishedRepoUpdateParams struct {
	// when publishing, overwrite files in pool/ directory without notice
	ForceOverwrite bool `                         json:"ForceOverwrite"  example:"false"`
	// GPG options
	Signing signingParams `                       json:"Signing"`
	// Don't generate contents indexes
	SkipContents *bool `                          json:"SkipContents"    example:"false"`
	// Skip bz2 compression for index files
	SkipBz2 *bool `                               json:"SkipBz2"         example:"false"`
	// Don't remove unreferenced files in prefix/component
	SkipCleanup *bool `                           json:"SkipCleanup"     example:"false"`
	// Provide index files by hash
	AcquireByHash *bool `                         json:"AcquireByHash"   example:"false"`
	// Enable multiple packages with the same filename in different distributions
	MultiDist *bool `                             json:"MultiDist"       example:"false"`
}

// @Summary Update Published Repository
// @Description **Update a published repository**
// @Description
// @Description Publish pending source component changes which were added with `Add/Remove/Replace Source Components`
// @Description
// @Description See also: `aptly publish update`
// @Tags Publish
// @Param prefix path string true "publishing prefix"
// @Param distribution path string true "distribution name"
// @Param _async query bool false "Run in background and return task object"
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

		err = published.Publish(context.PackagePool(), context, collectionFactory, signer, out, b.ForceOverwrite, context.SkelPath())
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
