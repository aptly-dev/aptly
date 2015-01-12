package api

import (
	"fmt"
	"sort"
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/http"
	"github.com/smira/aptly/query"
	"github.com/smira/aptly/utils"
)

func getVerifier(ignoreSignatures bool, keyRings []string) (utils.Verifier, error) {
	if ignoreSignatures {
		return nil, nil
	}

	verifier := &utils.GpgVerifier{}
	for _, keyRing := range keyRings {
		verifier.AddKeyring(keyRing)
	}

	err := verifier.InitKeyring()
	if err != nil {
		return nil, err
	}

	return verifier, nil
}

// GET /api/mirrors
func apiMirrorsList(c *gin.Context) {
	collection := context.CollectionFactory().RemoteRepoCollection()
	collection.Lock()
	defer collection.Unlock()

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
		Name                string `binding:"required"`
		ArchiveURL          string `binding:"required"`
		Distribution        string
		Components          []string
		Architectures       []string
		DownloadSources     bool
		DownloadUdebs       bool
		Filter              string
		FilterWithDeps      bool
		SkipComponentCheck  bool
		IgnoreSignatures    bool
		Keyrings            []string
	}

	b.DownloadSources = context.Config().DownloadSourcePackages
	b.IgnoreSignatures = context.Config().GpgDisableVerify
	b.Architectures = context.ArchitecturesList()

	if !c.Bind(&b) {
		return
	}

	collection := context.CollectionFactory().RemoteRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	if strings.HasPrefix(b.ArchiveURL, "ppa:") {
		b.ArchiveURL, b.Distribution, b.Components, err = deb.ParsePPA(b.ArchiveURL, context.Config())
		if err != nil {
			c.Fail(400, err)
			return
		}
	}

	if b.Filter != "" {
		_, err = query.Parse(b.Filter)
		if err != nil {
			c.Fail(400, fmt.Errorf("unable to create mirror: %s", err))
			return
		}
	}

	repo, err := deb.NewRemoteRepo(b.Name, b.ArchiveURL, b.Distribution, b.Components, b.Architectures,
		b.DownloadSources, b.DownloadUdebs)

	if err != nil {
		c.Fail(400, fmt.Errorf("unable to create mirror: %s", err))
		return
	}

	repo.Filter = b.Filter
	repo.FilterWithDeps = b.FilterWithDeps
	repo.SkipComponentCheck = b.SkipComponentCheck
	repo.DownloadSources = b.DownloadSources
	repo.DownloadUdebs = b.DownloadUdebs

	verifier, err := getVerifier(b.IgnoreSignatures, b.Keyrings)
	if err != nil {
		c.Fail(400, fmt.Errorf("unable to initialize GPG verifier: %s", err))
		return
	}

	err = repo.Fetch(context.Downloader(), verifier)
	if err != nil {
		c.Fail(403, fmt.Errorf("unable to fetch mirror: %s", err))
		return
	}

	err = collection.Add(repo)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to add mirror: %s", err))
		return
	}

	c.JSON(201, repo)
}

// DELETE /api/mirrors/:name
func apiMirrorsDrop(c *gin.Context) {
	name := c.Params.ByName("name")
	force := c.Request.URL.Query().Get("force") == "1"

	mirrorCollection := context.CollectionFactory().RemoteRepoCollection()
	mirrorCollection.Lock()
	defer mirrorCollection.Unlock()

	snapshotCollection := context.CollectionFactory().SnapshotCollection()
	snapshotCollection.Lock()
	defer snapshotCollection.Unlock()

	repo, err := mirrorCollection.ByName(name)
	if err != nil {
		c.Fail(404, fmt.Errorf("unable to drop: %s", err))
		return
	}

	err = repo.CheckLock()
	if err != nil {
		c.Fail(409, fmt.Errorf("unable to drop: %s", err))
		return
	}

	if !force {
		snapshots := snapshotCollection.ByRemoteRepoSource(repo)

		if len(snapshots) > 0 {
			c.Fail(409, fmt.Errorf("won't delete mirror with snapshots, use 'force=1' to override"))
			return
		}
	}

	err = mirrorCollection.Drop(repo)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to drop: %s", err))
		return
	}
}

// GET /api/mirrors/:name
func apiMirrorsShow(c *gin.Context) {
	collection := context.CollectionFactory().RemoteRepoCollection()
	collection.RLock()
	defer collection.RUnlock()

	name := c.Params.ByName("name")
	repo, err := collection.ByName(name)
	if err != nil {
		c.Fail(404, fmt.Errorf("unable to show: %s", err))
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to show: %s", err))
	}

	c.JSON(200, repo)
}

// GET /api/mirrors/:name/packages
func apiMirrorsPackages(c *gin.Context) {
	collection := context.CollectionFactory().RemoteRepoCollection()
	collection.RLock()
	defer collection.RUnlock()

	name := c.Params.ByName("name")
	repo, err := collection.ByName(name)
	if err != nil {
		c.Fail(404, fmt.Errorf("unable to show: %s", err))
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to show: %s", err))
	}

	if repo.LastDownloadDate.IsZero() {
		c.Fail(403, fmt.Errorf("Unable to show package list, mirror hasn't been downloaded yet."))
		return
	} else {
		reflist := repo.RefList()
		result := []*deb.Package{}

		list, err := deb.NewPackageListFromRefList(reflist, context.CollectionFactory().PackageCollection(), context.Progress())
		if err != nil {
			c.Fail(404, err)
			return
		}

		queryS := c.Request.URL.Query().Get("q")
		if queryS != "" {
			q, err := query.Parse(c.Request.URL.Query().Get("q"))
			if err != nil {
				c.Fail(400, err)
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

			list, err = list.Filter([]deb.PackageQuery{q}, withDeps,
				nil, context.DependencyOptions(), architecturesList)
			if err != nil {
				c.Fail(500, fmt.Errorf("unable to search: %s", err))
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
}

// PUT /api/mirrors/:name
func apiMirrorsUpdate(c *gin.Context) {
	var (
		err      error
		remote  *deb.RemoteRepo
	)

	var b struct {
		Name                string
		Filter              string
		FilterWithDeps      bool
		ForceComponents     bool
		DownloadSources     bool
		DownloadUdebs       bool
		Architectures     []string
		Components        []string
		SkipComponentCheck  bool
		IgnoreSignatures    bool
		Keyrings          []string
		ForceUpdate         bool
		DownloadLimit       int64
	}

	collection := context.CollectionFactory().RemoteRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	remote, err = collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	b.Name = remote.Name
	b.DownloadUdebs = remote.DownloadUdebs
	b.DownloadSources = remote.DownloadSources
	b.SkipComponentCheck = remote.SkipComponentCheck
	b.FilterWithDeps = remote.FilterWithDeps
	b.Filter = remote.Filter
	b.Architectures = remote.Architectures
	b.Components = remote.Components

	if !c.Bind(&b) {
		return
	}

	if b.Name != remote.Name {
		_, err = collection.ByName(b.Name)
		if err == nil {
			c.Fail(409, fmt.Errorf("unable to rename: mirror %s already exists", b.Name))
			return
		}
	}

	if b.DownloadUdebs != remote.DownloadUdebs {
		if remote.IsFlat() && b.DownloadUdebs {
			c.Fail(400, fmt.Errorf("unable to update: flat mirrors don't support udebs"))
			return
		}
	}

	remote.Name = b.Name
	remote.DownloadUdebs = b.DownloadUdebs
	remote.DownloadSources = b.DownloadSources
	remote.SkipComponentCheck = b.SkipComponentCheck
	remote.FilterWithDeps = b.FilterWithDeps
	remote.Filter = b.Filter
	remote.Architectures = b.Architectures
	remote.Components = b.Components

	verifier, err := getVerifier(b.IgnoreSignatures, b.Keyrings)
	if err != nil {
		c.Fail(400, fmt.Errorf("unable to initialize GPG verifier: %s", err))
		return
	}

	downloader := http.NewDownloader(context.Config().DownloadConcurrency, b.DownloadLimit*1024, context.Progress())
	err = remote.Fetch(downloader, verifier)
	if err != nil {
		c.Fail(400, fmt.Errorf("unable to update: %s", err))
		return
	}

	if !b.ForceUpdate {
		err = remote.CheckLock()
		if err != nil {
			c.Fail(409, fmt.Errorf("unable to update: %s", err))
			return
		}
	}

	err = remote.DownloadPackageIndexes(context.Progress(), downloader, context.CollectionFactory(), b.SkipComponentCheck)
	if err != nil {
		c.Fail(400, fmt.Errorf("unable to update: %s", err))
		return
	}

	if remote.Filter != "" {
		var filterQuery deb.PackageQuery

		filterQuery, err = query.Parse(remote.Filter)
		if err != nil {
			c.Fail(400, fmt.Errorf("unable to update: %s", err))
			return
		}

		_, _, err = remote.ApplyFilter(context.DependencyOptions(), filterQuery)
		if err != nil {
			c.Fail(400, fmt.Errorf("unable to update: %s", err))
			return
		}
	}

	queue, _, err := remote.BuildDownloadQueue(context.PackagePool())
	if err != nil {
		c.Fail(400, fmt.Errorf("unable to update: %s", err))
		return
	}

	remote.MarkAsUpdating()
	err = collection.Update(remote)
	if err != nil {
		c.Fail(400, fmt.Errorf("unable to update: %s", err))
		return
	}

	// In separate goroutine (to avoid blocking main), push queue to downloader
	ch := make(chan error, len(queue))
	go func() {
		defer func() {
			remote.MarkAsIdle()
			collection.Update(remote)
		}()

		for _, task := range queue {
			context.Downloader().DownloadWithChecksum(remote.PackageURL(task.RepoURI).String(), task.DestinationPath, ch, task.Checksums, b.SkipComponentCheck)
		}

        remote.FinalizeDownload()
		queue = nil
	}()

	c.JSON(200, remote)
}
