package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/smira/aptly/deb"
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
		Comment             string
		DefaultDistribution string
		DefaultComponent    string
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

	if b.Comment != "" {
		repo.Comment = b.Comment
	}
	if b.DefaultDistribution != "" {
		repo.DefaultDistribution = b.DefaultDistribution
	}
	if b.DefaultComponent != "" {
		repo.DefaultComponent = b.DefaultComponent
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
	var b struct {
		Force bool
	}

	if !c.Bind(&b) {
		return
	}

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

	snapshots := snapshotCollection.ByLocalRepoSource(repo)
	if len(snapshots) > 0 {
		c.Fail(409, fmt.Errorf("unable to drop, local repo has snapshots, use Force to override"))
		return
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

	c.JSON(200, repo.RefList().Strings())
}

// POST /repos/:name/packages
func apiReposPackagesAdd(c *gin.Context) {

}
