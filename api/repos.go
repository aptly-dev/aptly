package api

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
)

// GET /api/repos
func apiReposList(c *gin.Context) {
	result := []*deb.LocalRepo{}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()
	collection.ForEach(func(r *deb.LocalRepo) error {
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

	if c.Bind(&b) != nil {
		return
	}

	repo := deb.NewLocalRepo(b.Name, b.Comment)
	repo.DefaultComponent = b.DefaultComponent
	repo.DefaultDistribution = b.DefaultDistribution

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()
	err := collection.Add(repo)
	if err != nil {
		c.AbortWithError(400, err)
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

	if c.Bind(&b) != nil {
		return
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
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
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, repo)
}

// GET /api/repos/:name
func apiReposShow(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	c.JSON(200, repo)
}

// DELETE /api/repos/:name
func apiReposDrop(c *gin.Context) {
	force := c.Request.URL.Query().Get("force") == "1"

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()
	snapshotCollection := collectionFactory.SnapshotCollection()
	publishedCollection := collectionFactory.PublishedRepoCollection()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	published := publishedCollection.ByLocalRepo(repo)
	if len(published) > 0 {
		c.AbortWithError(409, fmt.Errorf("unable to drop, local repo is published"))
		return
	}

	if !force {
		snapshots := snapshotCollection.ByLocalRepoSource(repo)
		if len(snapshots) > 0 {
			c.AbortWithError(409, fmt.Errorf("unable to drop, local repo has snapshots, use ?force=1 to override"))
			return
		}
	}

	err = collection.Drop(repo)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.JSON(200, gin.H{})
}

// GET /api/repos/:name/packages
func apiReposPackagesShow(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	showPackages(c, repo.RefList(), collectionFactory)
}

// Handler for both add and delete
func apiReposPackagesAddDelete(c *gin.Context, cb func(list *deb.PackageList, p *deb.Package) error) {
	var b struct {
		PackageRefs []string
	}

	if c.Bind(&b) != nil {
		return
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	list, err := deb.NewPackageListFromRefList(repo.RefList(), collectionFactory.PackageCollection(), nil)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

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
		err = cb(list, p)
		if err != nil {
			c.AbortWithError(400, err)
			return
		}
	}

	repo.UpdateRefList(deb.NewPackageRefListFromPackageList(list))

	err = collectionFactory.LocalRepoCollection().Update(repo)
	if err != nil {
		c.AbortWithError(500, fmt.Errorf("unable to save: %s", err))
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
		c.AbortWithError(400, fmt.Errorf("wrong file"))
		return
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()

	repo, err := collection.ByName(c.Params.ByName("name"))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	verifier := context.GetVerifier()

	var (
		sources                      []string
		packageFiles, failedFiles    []string
		otherFiles                   []string
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

	packageFiles, otherFiles, failedFiles = deb.CollectPackageFiles(sources, reporter)

	list, err = deb.NewPackageListFromRefList(repo.RefList(), collectionFactory.PackageCollection(), nil)
	if err != nil {
		c.AbortWithError(500, fmt.Errorf("unable to load packages: %s", err))
		return
	}

	processedFiles, failedFiles2, err = deb.ImportPackageFiles(list, packageFiles, forceReplace, verifier, context.PackagePool(),
		collectionFactory.PackageCollection(), reporter, nil, collectionFactory.ChecksumCollection)
	failedFiles = append(failedFiles, failedFiles2...)

	processedFiles = append(processedFiles, otherFiles...)

	if err != nil {
		c.AbortWithError(500, fmt.Errorf("unable to import package files: %s", err))
		return
	}

	repo.UpdateRefList(deb.NewPackageRefListFromPackageList(list))

	err = collectionFactory.LocalRepoCollection().Update(repo)
	if err != nil {
		c.AbortWithError(500, fmt.Errorf("unable to save: %s", err))
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

// POST /repos/:name/include/:dir/:file
func apiReposIncludePackageFromFile(c *gin.Context) {
	// redirect all work to dir method
	apiReposIncludePackageFromDir(c)
}

// POST /repos/:name/include/:dir
func apiReposIncludePackageFromDir(c *gin.Context) {
	forceReplace := c.Request.URL.Query().Get("forceReplace") == "1"
	noRemoveFiles := c.Request.URL.Query().Get("noRemoveFiles") == "1"
	acceptUnsigned := c.Request.URL.Query().Get("acceptUnsigned") == "1"
	ignoreSignature := c.Request.URL.Query().Get("ignoreSignature") == "1"

	repoTemplateString := c.Params.ByName("name")

	if !verifyDir(c) {
		return
	}

	fileParam := c.Params.ByName("file")
	if fileParam != "" && !verifyPath(fileParam) {
		c.AbortWithError(400, fmt.Errorf("wrong file"))
		return
	}

	var (
		err                       error
		verifier                  = context.GetVerifier()
		sources, changesFiles     []string
		failedFiles, failedFiles2 []string
		reporter                  = &aptly.RecordingResultReporter{
			Warnings:     []string{},
			AddedLines:   []string{},
			RemovedLines: []string{},
		}
	)

	if fileParam == "" {
		sources = []string{filepath.Join(context.UploadPath(), c.Params.ByName("dir"))}
	} else {
		sources = []string{filepath.Join(context.UploadPath(), c.Params.ByName("dir"), c.Params.ByName("file"))}
	}

	collectionFactory := context.NewCollectionFactory()
	changesFiles, failedFiles = deb.CollectChangesFiles(sources, reporter)
	_, failedFiles2, err = deb.ImportChangesFiles(
		changesFiles, reporter, acceptUnsigned, ignoreSignature, forceReplace, noRemoveFiles, verifier,
		repoTemplateString, context.Progress(), collectionFactory.LocalRepoCollection(), collectionFactory.PackageCollection(),
		context.PackagePool(), collectionFactory.ChecksumCollection, nil, query.Parse)
	failedFiles = append(failedFiles, failedFiles2...)

	if err != nil {
		c.AbortWithError(500, fmt.Errorf("unable to import changes files: %s", err))
		return
	}

	if !noRemoveFiles {
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
