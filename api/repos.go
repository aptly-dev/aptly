package api

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/utils"
)

// GET /api/repos
func apiReposList(c *gin.Context) {
	result := []*deb.LocalRepo{}

	collection := context.CollectionFactory().LocalRepoCollection()
	collection.RLock()
	defer collection.RUnlock()

	context.CollectionFactory().LocalRepoCollection().ForEach(func(r *deb.LocalRepo) error {
		result = append(result, r)
		return nil
	})

	c.JSON(200, result)
}

// POST /api/repos
func apiReposCreate(c *gin.Context) {
	var b struct {
		Name                string `binding:"required"`
		Comment             string
		DefaultDistribution string
		DefaultComponent    string
	}

	if !c.Bind(&b) {
		return
	}

	repo := deb.NewLocalRepo(b.Name, b.Comment)
	repo.DefaultComponent = b.DefaultComponent
	repo.DefaultDistribution = b.DefaultDistribution

	collection := context.CollectionFactory().LocalRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	err := context.CollectionFactory().LocalRepoCollection().Add(repo)
	if err != nil {
		c.Fail(400, err)
		return
	}

	c.JSON(201, repo)
}

// PUT /api/repos/:name
func apiReposEdit(c *gin.Context) {
	var b struct {
		Comment             *string
		DefaultDistribution *string
		DefaultComponent    *string
	}

	if !c.Bind(&b) {
		return
	}

	collection := context.CollectionFactory().LocalRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	if b.Comment != nil {
		repo.Comment = *b.Comment
	}
	if b.DefaultDistribution != nil {
		repo.DefaultDistribution = *b.DefaultDistribution
	}
	if b.DefaultComponent != nil {
		repo.DefaultComponent = *b.DefaultComponent
	}

	err = collection.Update(repo)
	if err != nil {
		c.Fail(500, err)
		return
	}

	c.JSON(200, repo)
}

// GET /api/repos/:name
func apiReposShow(c *gin.Context) {
	collection := context.CollectionFactory().LocalRepoCollection()
	collection.RLock()
	defer collection.RUnlock()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	c.JSON(200, repo)
}

// DELETE /api/repos/:name
func apiReposDrop(c *gin.Context) {
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

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	published := publishedCollection.ByLocalRepo(repo)
	if len(published) > 0 {
		c.Fail(409, fmt.Errorf("unable to drop, local repo is published"))
		return
	}

	if !force {
		snapshots := snapshotCollection.ByLocalRepoSource(repo)
		if len(snapshots) > 0 {
			c.Fail(409, fmt.Errorf("unable to drop, local repo has snapshots, use ?force=1 to override"))
			return
		}
	}

	err = collection.Drop(repo)
	if err != nil {
		c.Fail(500, err)
		return
	}

	c.JSON(200, gin.H{})
}

// GET /api/repos/:name/packages
func apiReposPackagesShow(c *gin.Context) {
	collection := context.CollectionFactory().LocalRepoCollection()
	collection.RLock()
	defer collection.RUnlock()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.Fail(500, err)
		return
	}

	showPackages(c, repo.RefList())
}

// Handler for both add and delete
func apiReposPackagesAddDelete(c *gin.Context, cb func(list *deb.PackageList, p *deb.Package) error) {
	var b struct {
		PackageRefs []string
	}

	if !c.Bind(&b) {
		return
	}

	collection := context.CollectionFactory().LocalRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.Fail(500, err)
		return
	}

	list, err := deb.NewPackageListFromRefList(repo.RefList(), context.CollectionFactory().PackageCollection(), nil)
	if err != nil {
		c.Fail(500, err)
		return
	}

	// verify package refs and build package list
	for _, ref := range b.PackageRefs {
		var p *deb.Package

		p, err = context.CollectionFactory().PackageCollection().ByKey([]byte(ref))
		if err != nil {
			if err == database.ErrNotFound {
				c.Fail(404, fmt.Errorf("package %s: %s", ref, err))
			} else {
				c.Fail(500, err)
			}
			return
		}
		err = cb(list, p)
		if err != nil {
			c.Fail(400, err)
			return
		}
	}

	repo.UpdateRefList(deb.NewPackageRefListFromPackageList(list))

	err = context.CollectionFactory().LocalRepoCollection().Update(repo)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to save: %s", err))
		return
	}

	c.JSON(200, repo)

}

// POST /repos/:name/packages
func apiReposPackagesAdd(c *gin.Context) {
	apiReposPackagesAddDelete(c, func(list *deb.PackageList, p *deb.Package) error {
		return list.Add(p)
	})
}

// DELETE /repos/:name/packages
func apiReposPackagesDelete(c *gin.Context) {
	apiReposPackagesAddDelete(c, func(list *deb.PackageList, p *deb.Package) error {
		list.Remove(p)
		return nil
	})
}

// POST /repos/:name/file/:dir/:file
func apiReposPackageFromFile(c *gin.Context) {
	// redirect all work to dir method
	apiReposPackageFromDir(c)
}

// POST /repos/:name/file/:dir
func apiReposPackageFromDir(c *gin.Context) {
	forceReplace := c.Request.URL.Query().Get("forceReplace") == "1"
	noRemove := c.Request.URL.Query().Get("noRemove") == "1"

	if !verifyDir(c) {
		return
	}

	fileParam := c.Params.ByName("file")
	if fileParam != "" && !verifyPath(fileParam) {
		c.Fail(400, fmt.Errorf("wrong file"))
		return
	}

	collection := context.CollectionFactory().LocalRepoCollection()
	collection.Lock()
	defer collection.Unlock()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.Fail(404, err)
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.Fail(500, err)
		return
	}

	verifier := context.GetVerifier()

	var (
		sources                      []string
		packageFiles, failedFiles    []string
		processedFiles, failedFiles2 []string
		reporter                     = &aptly.RecordingResultReporter{
			Warnings:     []string{},
			AddedLines:   []string{},
			RemovedLines: []string{},
		}
		list *deb.PackageList
	)

	if fileParam == "" {
		sources = []string{filepath.Join(context.UploadPath(), c.Params.ByName("dir"))}
	} else {
		sources = []string{filepath.Join(context.UploadPath(), c.Params.ByName("dir"), c.Params.ByName("file"))}
	}

	packageFiles, failedFiles = deb.CollectPackageFiles(sources, reporter)

	list, err = deb.NewPackageListFromRefList(repo.RefList(), context.CollectionFactory().PackageCollection(), nil)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to load packages: %s", err))
		return
	}

	processedFiles, failedFiles2, err = deb.ImportPackageFiles(list, packageFiles, forceReplace, verifier, context.PackagePool(),
		context.CollectionFactory().PackageCollection(), reporter, nil, context.CollectionFactory().ChecksumCollection())
	failedFiles = append(failedFiles, failedFiles2...)

	if err != nil {
		c.Fail(500, fmt.Errorf("unable to import package files: %s", err))
		return
	}

	repo.UpdateRefList(deb.NewPackageRefListFromPackageList(list))

	err = context.CollectionFactory().LocalRepoCollection().Update(repo)
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to save: %s", err))
		return
	}

	if !noRemove {
		processedFiles = utils.StrSliceDeduplicate(processedFiles)

		for _, file := range processedFiles {
			err := os.Remove(file)
			if err != nil {
				reporter.Warning("unable to remove file %s: %s", file, err)
			}
		}

		// atempt to remove dir, if it fails, that's fine: probably it's not empty
		os.Remove(filepath.Join(context.UploadPath(), c.Params.ByName("dir")))
	}

	if failedFiles == nil {
		failedFiles = []string{}
	}

	c.JSON(200, gin.H{
		"Report":      reporter,
		"FailedFiles": failedFiles,
	})
}
