package api

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/query"
	"github.com/aptly-dev/aptly/task"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func getVerifier(keyRings []string) (pgp.Verifier, error) {
	verifier := context.GetVerifier()
	for _, keyRing := range keyRings {
		verifier.AddKeyring(keyRing)
	}

	err := verifier.InitKeyring(false)
	if err != nil {
		return nil, err
	}

	return verifier, nil
}

// GET /api/mirrors
func apiMirrorsList(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.RemoteRepoCollection()

	result := []*deb.RemoteRepo{}
	collection.ForEach(func(repo *deb.RemoteRepo) error {
		result = append(result, repo)
		return nil
	})

	c.JSON(200, result)
}

// POST /api/mirrors
func apiMirrorsCreate(c *gin.Context) {
	var err error
	var b struct {
		Name                  string `binding:"required"`
		ArchiveURL            string `binding:"required"`
		Distribution          string
		Filter                string
		Components            []string
		Architectures         []string
		Keyrings              []string
		DownloadSources       bool
		DownloadUdebs         bool
		DownloadInstaller     bool
		FilterWithDeps        bool
		SkipComponentCheck    bool
		SkipArchitectureCheck bool
		IgnoreSignatures      bool
	}

	b.DownloadSources = context.Config().DownloadSourcePackages
	b.IgnoreSignatures = context.Config().GpgDisableVerify
	b.Architectures = context.ArchitecturesList()

	if c.Bind(&b) != nil {
		return
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.RemoteRepoCollection()

	if strings.HasPrefix(b.ArchiveURL, "ppa:") {
		b.ArchiveURL, b.Distribution, b.Components, err = deb.ParsePPA(b.ArchiveURL, context.Config())
		if err != nil {
			AbortWithJSONError(c, 400, err)
			return
		}
	}

	if b.Filter != "" {
		_, err = query.Parse(b.Filter)
		if err != nil {
			AbortWithJSONError(c, 400, fmt.Errorf("unable to create mirror: %s", err))
			return
		}
	}

	repo, err := deb.NewRemoteRepo(b.Name, b.ArchiveURL, b.Distribution, b.Components, b.Architectures,
		b.DownloadSources, b.DownloadUdebs, b.DownloadInstaller)

	if err != nil {
		AbortWithJSONError(c, 400, fmt.Errorf("unable to create mirror: %s", err))
		return
	}

	repo.Filter = b.Filter
	repo.FilterWithDeps = b.FilterWithDeps
	repo.SkipComponentCheck = b.SkipComponentCheck
	repo.SkipArchitectureCheck = b.SkipArchitectureCheck
	repo.DownloadSources = b.DownloadSources
	repo.DownloadUdebs = b.DownloadUdebs

	verifier, err := getVerifier(b.Keyrings)
	if err != nil {
		AbortWithJSONError(c, 400, fmt.Errorf("unable to initialize GPG verifier: %s", err))
		return
	}

	downloader := context.NewDownloader(nil)
	err = repo.Fetch(downloader, verifier, b.IgnoreSignatures)
	if err != nil {
		AbortWithJSONError(c, 400, fmt.Errorf("unable to fetch mirror: %s", err))
		return
	}

	err = collection.Add(repo)
	if err != nil {
		AbortWithJSONError(c, 500, fmt.Errorf("unable to add mirror: %s", err))
		return
	}

	c.JSON(201, repo)
}

// DELETE /api/mirrors/:name
func apiMirrorsDrop(c *gin.Context) {
	name := c.Params.ByName("name")
	force := c.Request.URL.Query().Get("force") == "1"

	collectionFactory := context.NewCollectionFactory()
	mirrorCollection := collectionFactory.RemoteRepoCollection()
	snapshotCollection := collectionFactory.SnapshotCollection()

	repo, err := mirrorCollection.ByName(name)
	if err != nil {
		AbortWithJSONError(c, 404, fmt.Errorf("unable to drop: %s", err))
		return
	}

	resources := []string{string(repo.Key())}
	taskName := fmt.Sprintf("Delete mirror %s", name)
	maybeRunTaskInBackground(c, taskName, resources, func(_ aptly.Progress, _ *task.Detail) (*task.ProcessReturnValue, error) {
		err := repo.CheckLock()
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to drop: %v", err)
		}

		if !force {
			snapshots := snapshotCollection.ByRemoteRepoSource(repo)

			if len(snapshots) > 0 {
				return &task.ProcessReturnValue{Code: http.StatusForbidden, Value: nil}, fmt.Errorf("won't delete mirror with snapshots, use 'force=1' to override")
			}
		}

		err = mirrorCollection.Drop(repo)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to drop: %v", err)
		}
		return &task.ProcessReturnValue{Code: http.StatusNoContent, Value: nil}, nil
	})
}

// GET /api/mirrors/:name
func apiMirrorsShow(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.RemoteRepoCollection()

	name := c.Params.ByName("name")
	repo, err := collection.ByName(name)
	if err != nil {
		AbortWithJSONError(c, 404, fmt.Errorf("unable to show: %s", err))
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		AbortWithJSONError(c, 500, fmt.Errorf("unable to show: %s", err))
	}

	c.JSON(200, repo)
}

// GET /api/mirrors/:name/packages
func apiMirrorsPackages(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.RemoteRepoCollection()

	name := c.Params.ByName("name")
	repo, err := collection.ByName(name)
	if err != nil {
		AbortWithJSONError(c, 404, fmt.Errorf("unable to show: %s", err))
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		AbortWithJSONError(c, 500, fmt.Errorf("unable to show: %s", err))
	}

	if repo.LastDownloadDate.IsZero() {
		AbortWithJSONError(c, 404, fmt.Errorf("unable to show package list, mirror hasn't been downloaded yet"))
		return
	}

	reflist := repo.RefList()
	result := []*deb.Package{}

	list, err := deb.NewPackageListFromRefList(reflist, collectionFactory.PackageCollection(), nil)
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	queryS := c.Request.URL.Query().Get("q")
	if queryS != "" {
		q, err := query.Parse(c.Request.URL.Query().Get("q"))
		if err != nil {
			AbortWithJSONError(c, 400, err)
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
				AbortWithJSONError(c, 400, fmt.Errorf("unable to determine list of architectures, please specify explicitly"))
				return
			}
		}

		list.PrepareIndex()

		list, err = list.Filter([]deb.PackageQuery{q}, withDeps,
			nil, context.DependencyOptions(), architecturesList)
		if err != nil {
			AbortWithJSONError(c, 500, fmt.Errorf("unable to search: %s", err))
		}
	}

	if c.Request.URL.Query().Get("format") == "details" {
		list.ForEach(func(p *deb.Package) error {
			result = append(result, p)
			return nil
		})

		c.JSON(200, result)
	} else {
		c.JSON(200, list.Strings())
	}
}

// PUT /api/mirrors/:name
func apiMirrorsUpdate(c *gin.Context) {
	var (
		err    error
		remote *deb.RemoteRepo
	)

	var b struct {
		Name                  string
		ArchiveURL            string
		Filter                string
		Architectures         []string
		Components            []string
		Keyrings              []string
		FilterWithDeps        bool
		DownloadSources       bool
		DownloadUdebs         bool
		SkipComponentCheck    bool
		SkipArchitectureCheck bool
		IgnoreChecksums       bool
		IgnoreSignatures      bool
		ForceUpdate           bool
		SkipExistingPackages  bool
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.RemoteRepoCollection()

	remote, err = collection.ByName(c.Params.ByName("name"))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	b.Name = remote.Name
	b.DownloadUdebs = remote.DownloadUdebs
	b.DownloadSources = remote.DownloadSources
	b.SkipComponentCheck = remote.SkipComponentCheck
	b.SkipArchitectureCheck = remote.SkipArchitectureCheck
	b.FilterWithDeps = remote.FilterWithDeps
	b.Filter = remote.Filter
	b.Architectures = remote.Architectures
	b.Components = remote.Components
	b.IgnoreSignatures = context.Config().GpgDisableVerify

	log.Info().Msgf("%s: Starting mirror update", b.Name)

	if c.Bind(&b) != nil {
		return
	}

	if b.Name != remote.Name {
		_, err = collection.ByName(b.Name)
		if err == nil {
			AbortWithJSONError(c, 409, fmt.Errorf("unable to rename: mirror %s already exists", b.Name))
			return
		}
	}

	if b.DownloadUdebs != remote.DownloadUdebs {
		if remote.IsFlat() && b.DownloadUdebs {
			AbortWithJSONError(c, 400, fmt.Errorf("unable to update: flat mirrors don't support udebs"))
			return
		}
	}

	if b.ArchiveURL != "" {
		remote.SetArchiveRoot(b.ArchiveURL)
	}

	remote.Name = b.Name
	remote.DownloadUdebs = b.DownloadUdebs
	remote.DownloadSources = b.DownloadSources
	remote.SkipComponentCheck = b.SkipComponentCheck
	remote.SkipArchitectureCheck = b.SkipArchitectureCheck
	remote.FilterWithDeps = b.FilterWithDeps
	remote.Filter = b.Filter
	remote.Architectures = b.Architectures
	remote.Components = b.Components

	verifier, err := getVerifier(b.Keyrings)
	if err != nil {
		AbortWithJSONError(c, 400, fmt.Errorf("unable to initialize GPG verifier: %s", err))
		return
	}

	resources := []string{string(remote.Key())}
	maybeRunTaskInBackground(c, "Update mirror "+b.Name, resources, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {

		downloader := context.NewDownloader(out)
		err := remote.Fetch(downloader, verifier, b.IgnoreSignatures)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
		}

		if !b.ForceUpdate {
			err = remote.CheckLock()
			if err != nil {
				return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
			}
		}

		err = remote.DownloadPackageIndexes(out, downloader, verifier, collectionFactory, b.IgnoreSignatures, b.SkipComponentCheck)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
		}

		if remote.Filter != "" {
			var filterQuery deb.PackageQuery

			filterQuery, err = query.Parse(remote.Filter)
			if err != nil {
				return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
			}

			_, _, err = remote.ApplyFilter(context.DependencyOptions(), filterQuery, out)
			if err != nil {
				return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
			}
		}

		queue, downloadSize, err := remote.BuildDownloadQueue(context.PackagePool(), collectionFactory.PackageCollection(),
			collectionFactory.ChecksumCollection(nil), b.SkipExistingPackages)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
		}

		defer func() {
			// on any interruption, unlock the mirror
			e := context.ReOpenDatabase()
			if e == nil {
				remote.MarkAsIdle()
				collection.Update(remote)
			}
		}()

		remote.MarkAsUpdating()
		err = collection.Update(remote)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
		}

		context.GoContextHandleSignals()

		count := len(queue)
		taskDetail := struct {
			TotalDownloadSize         int64
			RemainingDownloadSize     int64
			TotalNumberOfPackages     int
			RemainingNumberOfPackages int
		}{
			downloadSize, downloadSize, count, count,
		}
		detail.Store(taskDetail)

		downloadQueue := make(chan int)
		taskFinished := make(chan *deb.PackageDownloadTask)

		var (
			errors  []string
			errLock sync.Mutex
		)

		pushError := func(err error) {
			errLock.Lock()
			errors = append(errors, err.Error())
			errLock.Unlock()
		}

		go func() {
			for idx := range queue {
				select {
				case downloadQueue <- idx:
				case <-context.Done():
					return
				}
			}

			close(downloadQueue)
		}()

		// update of task details need to be done in order
		go func() {
			for {
				task, ok := <-taskFinished
				if !ok {
					return
				}

				taskDetail.RemainingDownloadSize -= task.File.Checksums.Size
				taskDetail.RemainingNumberOfPackages--
				detail.Store(taskDetail)
			}
		}()

		log.Info().Msgf("%s: Spawning background processes...", b.Name)
		var wg sync.WaitGroup
		for i := 0; i < context.Config().DownloadConcurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case idx, ok := <-downloadQueue:
						if !ok {
							return
						}

						task := &queue[idx]

						var e error

						// provision download location
						if pp, ok := context.PackagePool().(aptly.LocalPackagePool); ok {
							task.TempDownPath, e = pp.GenerateTempPath(task.File.Filename)
						} else {
							var file *os.File
							file, e = os.CreateTemp("", task.File.Filename)
							if e == nil {
								task.TempDownPath = file.Name()
								file.Close()
							}
						}
						if e != nil {
							pushError(e)
							continue
						}

						// download file...
						e = context.Downloader().DownloadWithChecksum(
							context,
							remote.PackageURL(task.File.DownloadURL()).String(),
							task.TempDownPath,
							&task.File.Checksums,
							b.IgnoreChecksums)
						if e != nil {
							pushError(e)
							continue
						}

						// and import it back to the pool
						task.File.PoolPath, err = context.PackagePool().Import(task.TempDownPath, task.File.Filename, &task.File.Checksums, true, collectionFactory.ChecksumCollection(nil))
						if err != nil {
							//return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to import file: %s", err)
							pushError(err)
							continue
						}

						// update "attached" files if any
						for _, additionalAtask := range task.Additional {
							additionalAtask.File.PoolPath = task.File.PoolPath
							additionalAtask.File.Checksums = task.File.Checksums
						}

						task.Done = true
						taskFinished <- task
					case <-context.Done():
						return
					}

				}
			}()
		}

		// Wait for all download goroutines to finish
		log.Info().Msgf("%s: Waiting for background processes to finish...", b.Name)
		wg.Wait()
		log.Info().Msgf("%s: Background processes finished", b.Name)
		close(taskFinished)

		defer func() {
			for _, task := range queue {
				if task.TempDownPath == "" {
					continue
				}

				if err := os.Remove(task.TempDownPath); err != nil && !os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "Failed to delete %s: %v\n", task.TempDownPath, err)
				}
			}
		}()

		select {
		case <-context.Done():
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: interrupted")
		default:
		}

		if len(errors) > 0 {
			log.Info().Msgf("%s: Unable to update because of previous errors", b.Name)
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: download errors:\n  %s", strings.Join(errors, "\n  "))
		}

		log.Info().Msgf("%s: Finalizing download...", b.Name)
		remote.FinalizeDownload(collectionFactory, out)
		err = collectionFactory.RemoteRepoCollection().Update(remote)
		if err != nil {
			return &task.ProcessReturnValue{Code: http.StatusInternalServerError, Value: nil}, fmt.Errorf("unable to update: %s", err)
		}

		log.Info().Msgf("%s: Mirror updated successfully", b.Name)
		return &task.ProcessReturnValue{Code: http.StatusNoContent, Value: nil}, nil
	})
}
