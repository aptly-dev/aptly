package api

import (
	"net/http"
	"os"
	"sync/atomic"

	"github.com/aptly-dev/aptly/aptly"
	ctx "github.com/aptly-dev/aptly/context"
	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var context *ctx.AptlyContext

func apiMetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		countPackagesByRepos()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}

// Router returns prebuilt with routes http.Handler
func Router(c *ctx.AptlyContext) http.Handler {
	if aptly.EnableDebug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	context = c

	router.UseRawPath = true

	if c.Config().LogFormat == "json" {
		c.StructuredLogging(true)
		utils.SetupJSONLogger(c.Config().LogLevel, os.Stdout)
		gin.DefaultWriter = utils.LogWriter{Logger: log.Logger}
		router.Use(JSONLogger())
	} else {
		c.StructuredLogging(false)
		utils.SetupDefaultLogger(c.Config().LogLevel)
		router.Use(gin.Logger())
	}

	router.Use(gin.Recovery(), gin.ErrorLogger())

	if c.Config().EnableMetricsEndpoint {
		MetricsCollectorRegistrar.Register(router)
	}

	if c.Config().ServeInAPIMode {
		router.GET("/repos/", reposListInAPIMode(c.Config().FileSystemPublishRoots))
		router.GET("/repos/:storage/*pkgPath", reposServeInAPIMode)
	}

	api := router.Group("/api")
	if context.Flags().Lookup("no-lock").Value.Get().(bool) {
		// We use a goroutine to count the number of
		// concurrent requests. When no more requests are
		// running, we close the database to free the lock.
		dbRequests = make(chan dbRequest)

		go acquireDatabase()

		api.Use(func(c *gin.Context) {
			var err error

			errCh := make(chan error)
			dbRequests <- dbRequest{acquiredb, errCh}

			err = <-errCh
			if err != nil {
				AbortWithJSONError(c, 500, err)
				return
			}

			defer func() {
				dbRequests <- dbRequest{releasedb, errCh}
				err = <-errCh
				if err != nil {
					AbortWithJSONError(c, 500, err)
				}
			}()

			c.Next()
		})
	}

	{
		if c.Config().EnableMetricsEndpoint {
			api.GET("/metrics", apiMetricsGet())
		}
		api.GET("/version", apiVersion)

		isReady := &atomic.Value{}
		isReady.Store(false)
		defer isReady.Store(true)
		api.GET("/ready", apiReady(isReady))
		api.GET("/healthy", apiHealthy)
	}

	{
		api.GET("/repos", apiReposList)
		api.POST("/repos", apiReposCreate)
		api.GET("/repos/:name", apiReposShow)
		api.PUT("/repos/:name", apiReposEdit)
		api.DELETE("/repos/:name", apiReposDrop)

		api.GET("/repos/:name/packages", apiReposPackagesShow)
		api.POST("/repos/:name/packages", apiReposPackagesAdd)
		api.DELETE("/repos/:name/packages", apiReposPackagesDelete)

		api.POST("/repos/:name/file/:dir/:file", apiReposPackageFromFile)
		api.POST("/repos/:name/file/:dir", apiReposPackageFromDir)

		api.POST("/repos/:name/include/:dir/:file", apiReposIncludePackageFromFile)
		api.POST("/repos/:name/include/:dir", apiReposIncludePackageFromDir)

		api.POST("/repos/:name/snapshots", apiSnapshotsCreateFromRepository)
	}

	{
		api.POST("/mirrors/:name/snapshots", apiSnapshotsCreateFromMirror)
	}

	{
		api.GET("/mirrors", apiMirrorsList)
		api.GET("/mirrors/:name", apiMirrorsShow)
		api.GET("/mirrors/:name/packages", apiMirrorsPackages)
		api.POST("/mirrors", apiMirrorsCreate)
		api.PUT("/mirrors/:name", apiMirrorsUpdate)
		api.DELETE("/mirrors/:name", apiMirrorsDrop)
	}

	{
		api.POST("/gpg/key", apiGPGAddKey)
	}

	{
		api.GET("/s3", apiS3List)
	}

	{
		api.GET("/files", apiFilesListDirs)
		api.POST("/files/:dir", apiFilesUpload)
		api.GET("/files/:dir", apiFilesListFiles)
		api.DELETE("/files/:dir", apiFilesDeleteDir)
		api.DELETE("/files/:dir/:name", apiFilesDeleteFile)
	}

	{
		api.GET("/publish", apiPublishList)
		api.POST("/publish", apiPublishRepoOrSnapshot)
		api.POST("/publish/:prefix", apiPublishRepoOrSnapshot)
		api.PUT("/publish/:prefix/:distribution", apiPublishUpdateSwitch)
		api.DELETE("/publish/:prefix/:distribution", apiPublishDrop)
	}

	{
		api.GET("/snapshots", apiSnapshotsList)
		api.POST("/snapshots", apiSnapshotsCreate)
		api.PUT("/snapshots/:name", apiSnapshotsUpdate)
		api.GET("/snapshots/:name", apiSnapshotsShow)
		api.GET("/snapshots/:name/packages", apiSnapshotsSearchPackages)
		api.DELETE("/snapshots/:name", apiSnapshotsDrop)
		api.GET("/snapshots/:name/diff/:withSnapshot", apiSnapshotsDiff)
		api.POST("/snapshots/merge", apiSnapshotsMerge)
	}

	{
		api.GET("/packages/:key", apiPackagesShow)
		api.GET("/packages", apiPackages)
	}

	{
		api.GET("/graph.:ext", apiGraph)
	}
	{
		api.POST("/db/cleanup", apiDbCleanup)
	}
	{
		api.GET("/tasks", apiTasksList)
		api.POST("/tasks-clear", apiTasksClear)
		api.GET("/tasks-wait", apiTasksWait)
		api.GET("/tasks/:id/wait", apiTasksWaitForTaskByID)
		api.GET("/tasks/:id/output", apiTasksOutputShow)
		api.GET("/tasks/:id/detail", apiTasksDetailShow)
		api.GET("/tasks/:id/return_value", apiTasksReturnValueShow)
		api.GET("/tasks/:id", apiTasksShow)
		api.DELETE("/tasks/:id", apiTasksDelete)
		api.POST("/tasks-dummy", apiTasksDummy)
	}

	return router
}
