package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
	"github.com/aptly-dev/aptly/task"
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
		Name                *string
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

	if b.Name != nil {
		_, err := collection.ByName(*b.Name)
		if err == nil {
			// already exists
			c.AbortWithError(404, err)
			return
		}
		repo.Name = *b.Name
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
	name := c.Params.ByName("name")

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()
	snapshotCollection := collectionFactory.SnapshotCollection()
	publishedCollection := collectionFactory.PublishedRepoCollection()

	repo, err := collection.ByName(name)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	resources := []string{string(repo.Key())}
	taskName := fmt.Sprintf("Delete repo %s", name)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, detail *task.Detail) (int, error) {
		published := publishedCollection.ByLocalRepo(repo)
		if len(published) > 0 {
			return http.StatusConflict, fmt.Errorf("unable to drop, local repo is published")
		}

		if !force {
			snapshots := snapshotCollection.ByLocalRepoSource(repo)
			if len(snapshots) > 0 {
				return http.StatusConflict, fmt.Errorf("unable to drop, local repo has snapshots, use ?force=1 to override")
			}
		}

		detail.Store(gin.H{})
		return http.StatusOK, collection.Drop(repo)
	})
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
func apiReposPackagesAddDelete(c *gin.Context, taskNamePrefix string, cb func(list *deb.PackageList, p *deb.Package, out aptly.Progress) error) {
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

	resources := []string{string(repo.Key())}
	maybeRunTaskInBackground(c, taskNamePrefix+repo.Name, resources, func(out aptly.Progress, detail *task.Detail) (int, error) {
		out.Printf("Loading packages...\n")
		list, err := deb.NewPackageListFromRefList(repo.RefList(), collectionFactory.PackageCollection(), nil)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		// verify package refs and build package list
		for _, ref := range b.PackageRefs {
			var p *deb.Package

			p, err = collectionFactory.PackageCollection().ByKey([]byte(ref))
			if err != nil {
				if err == database.ErrNotFound {
					return http.StatusNotFound, fmt.Errorf("packages %s: %s", ref, err)
				}

				return http.StatusInternalServerError, err
			}
			err = cb(list, p, out)
			if err != nil {
				return http.StatusBadRequest, err
			}
		}

		repo.UpdateRefList(deb.NewPackageRefListFromPackageList(list))

		err = collectionFactory.LocalRepoCollection().Update(repo)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("unable to save: %s", err)
		}
		detail.Store(repo)
		return http.StatusOK, nil
	})
}

// POST /repos/:name/packages
func apiReposPackagesAdd(c *gin.Context) {
	apiReposPackagesAddDelete(c, "Add packages to repo ", func(list *deb.PackageList, p *deb.Package, out aptly.Progress) error {
		out.Printf("Adding package %s\n", p.Name)
		return list.Add(p)
	})
}

// DELETE /repos/:name/packages
func apiReposPackagesDelete(c *gin.Context) {
	apiReposPackagesAddDelete(c, "Delete packages from repo ", func(list *deb.PackageList, p *deb.Package, out aptly.Progress) error {
		out.Printf("Removing package %s\n", p.Name)
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

	dirParam := c.Params.ByName("dir")
	fileParam := c.Params.ByName("file")
	if fileParam != "" && !verifyPath(fileParam) {
		c.AbortWithError(400, fmt.Errorf("wrong file"))
		return
	}

	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.LocalRepoCollection()

	name := c.Params.ByName("name")
	repo, err := collection.ByName(name)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	err = collection.LoadComplete(repo)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	var taskName string
	var sources []string
	if fileParam == "" {
		taskName = fmt.Sprintf("Add packages from dir %s to repo %s", dirParam, name)
		sources = []string{filepath.Join(context.UploadPath(), dirParam)}
	} else {
		sources = []string{filepath.Join(context.UploadPath(), dirParam, fileParam)}
		taskName = fmt.Sprintf("Add package %s from dir %s to repo %s", fileParam, dirParam, name)
	}

	resources := []string{string(repo.Key())}
	resources = append(resources, sources...)
	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, detail *task.Detail) (int, error) {
		verifier := context.GetVerifier()

		var (
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

		packageFiles, otherFiles, failedFiles = deb.CollectPackageFiles(sources, reporter)

		list, err := deb.NewPackageListFromRefList(repo.RefList(), collectionFactory.PackageCollection(), nil)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("unable to load packages: %s", err)
		}

		processedFiles, failedFiles2, err = deb.ImportPackageFiles(list, packageFiles, forceReplace, verifier, context.PackagePool(),
			collectionFactory.PackageCollection(), reporter, nil, collectionFactory.ChecksumCollection)
		failedFiles = append(failedFiles, failedFiles2...)
		processedFiles = append(processedFiles, otherFiles...)

		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("unable to import package files: %s", err)
		}

		repo.UpdateRefList(deb.NewPackageRefListFromPackageList(list))

		err = collectionFactory.LocalRepoCollection().Update(repo)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("unable to save: %s", err)
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
			os.Remove(filepath.Join(context.UploadPath(), dirParam))
		}

		if failedFiles == nil {
			failedFiles = []string{}
		}

		if len(reporter.AddedLines) > 0 {
			out.Printf("Added: %s\n", strings.Join(reporter.AddedLines, ", "))
		}
		if len(reporter.RemovedLines) > 0 {
			out.Printf("Removed: %s\n", strings.Join(reporter.RemovedLines, ", "))
		}
		if len(reporter.Warnings) > 0 {
			out.Printf("Warnings: %s\n", strings.Join(reporter.Warnings, ", "))
		}
		if len(failedFiles) > 0 {
			out.Printf("Failed files: %s\n", strings.Join(failedFiles, ", "))
		}

		detail.Store(gin.H{
			"Report":      reporter,
			"FailedFiles": failedFiles,
		})
		return http.StatusOK, nil
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
	collectionFactory := context.NewCollectionFactory()

	if !verifyDir(c) {
		return
	}

	var sources []string
	var taskName string
	dirParam := c.Params.ByName("dir")
	fileParam := c.Params.ByName("file")
	if fileParam != "" && !verifyPath(fileParam) {
		c.AbortWithError(400, fmt.Errorf("wrong file"))
		return
	}

	if fileParam == "" {
		taskName = fmt.Sprintf("Include packages from changes files in dir %s to repo matching template %s", dirParam, repoTemplateString)
		sources = []string{filepath.Join(context.UploadPath(), dirParam)}
	} else {
		taskName = fmt.Sprintf("Include packages from changes file %s from dir %s to repo matching template %s", fileParam, dirParam, repoTemplateString)
		sources = []string{filepath.Join(context.UploadPath(), dirParam, fileParam)}
	}

	repoTemplate, err := template.New("repo").Parse(repoTemplateString)
	if err != nil {
		c.AbortWithError(400, fmt.Errorf("error parsing repo template: %s", err))
		return
	}

	var resources []string
	if len(repoTemplate.Tree.Root.Nodes) > 1 {
		resources = append(resources, task.AllLocalReposResourcesKey)
	} else {
		// repo template string is simple text so only use resource key of specific repository
		repo, err := collectionFactory.LocalRepoCollection().ByName(repoTemplateString)
		if err != nil {
			c.AbortWithError(404, err)
			return
		}

		resources = append(resources, string(repo.Key()))
	}
	resources = append(resources, sources...)

	maybeRunTaskInBackground(c, taskName, resources, func(out aptly.Progress, detail *task.Detail) (int, error) {
		var (
			err                       error
			verifier                  = context.GetVerifier()
			changesFiles              []string
			failedFiles, failedFiles2 []string
			reporter                  = &aptly.RecordingResultReporter{
				Warnings:     []string{},
				AddedLines:   []string{},
				RemovedLines: []string{},
			}
		)

		changesFiles, failedFiles = deb.CollectChangesFiles(sources, reporter)
		_, failedFiles2, err = deb.ImportChangesFiles(
			changesFiles, reporter, acceptUnsigned, ignoreSignature, forceReplace, noRemoveFiles, verifier,
			repoTemplate, context.Progress(), collectionFactory.LocalRepoCollection(), collectionFactory.PackageCollection(),
			context.PackagePool(), collectionFactory.ChecksumCollection, nil, query.Parse)
		failedFiles = append(failedFiles, failedFiles2...)

		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("unable to import changes files: %s", err)
		}

		if !noRemoveFiles {
			// atempt to remove dir, if it fails, that's fine: probably it's not empty
			os.Remove(filepath.Join(context.UploadPath(), dirParam))
		}

		if failedFiles == nil {
			failedFiles = []string{}
		}

		if len(reporter.AddedLines) > 0 {
			out.Printf("Added: %s\n", strings.Join(reporter.AddedLines, ", "))
		}
		if len(reporter.RemovedLines) > 0 {
			out.Printf("Removed: %s\n", strings.Join(reporter.RemovedLines, ", "))
		}
		if len(reporter.Warnings) > 0 {
			out.Printf("Warnings: %s\n", strings.Join(reporter.Warnings, ", "))
		}
		if len(failedFiles) > 0 {
			out.Printf("Failed files: %s\n", strings.Join(failedFiles, ", "))
		}

		detail.Store(gin.H{
			"Report":      reporter,
			"FailedFiles": failedFiles,
		})
		return http.StatusOK, nil

	})
}
