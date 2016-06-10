package api

import (
	"net/http"

	"github.com/DanielHeckrath/gin-prometheus"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	ctx "github.com/smira/aptly/context"
)

func prometheusHandler() gin.HandlerFunc {
	h := prometheus.UninstrumentedHandler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

var context *ctx.AptlyContext

// Router returns prebuilt with routes http.Handler
func Router(c *ctx.AptlyContext) http.Handler {
	context = c

	router := gin.Default()
	router.Use(gin.ErrorLogger())

	if context.Flags().Lookup("no-lock").Value.Get().(bool) {
		// We use a goroutine to count the number of
		// concurrent requests. When no more requests are
		// running, we close the database to free the lock.
		requests := make(chan int)
		acks := make(chan error)

		go acquireDatabase(requests, acks)
		go cacheFlusher(requests, acks)

		router.Use(func(c *gin.Context) {
			requests <- ACQUIREDB
			err := <-acks
			if err != nil {
				c.Fail(500, err)
				return
			}
			defer func() {
				requests <- RELEASEDB
				err = <-acks
				if err != nil {
					c.Fail(500, err)
					return
				}
			}()
			c.Next()
		})

	} else {
		go cacheFlusher(nil, nil)
	}

	router.GET("/metrics", prometheusHandler())
	root := router.Group("/api")

	{
		root.GET("/version", ginprom.InstrumentHandlerFunc("version", apiVersion))
	}

	{
		root.GET("/repos", ginprom.InstrumentHandlerFunc("reposList", apiReposList))
		root.POST("/repos", ginprom.InstrumentHandlerFunc("reposCreate", apiReposCreate))
		root.GET("/repos/:name", ginprom.InstrumentHandlerFunc("reposShow", apiReposShow))
		root.PUT("/repos/:name", ginprom.InstrumentHandlerFunc("reposEdit", apiReposEdit))
		root.DELETE("/repos/:name", ginprom.InstrumentHandlerFunc("reposDrop", apiReposDrop))

		root.GET("/repos/:name/packages", ginprom.InstrumentHandlerFunc("reposPackagesShow", apiReposPackagesShow))
		root.POST("/repos/:name/packages", ginprom.InstrumentHandlerFunc("reposPackagesAdd", apiReposPackagesAdd))
		root.DELETE("/repos/:name/packages", ginprom.InstrumentHandlerFunc("reposPackagesDelete", apiReposPackagesDelete))

		root.POST("/repos/:name/file/:dir/:file", ginprom.InstrumentHandlerFunc("reposPackageFromFile", apiReposPackageFromFile))
		root.POST("/repos/:name/file/:dir", ginprom.InstrumentHandlerFunc("reposPackageFromDir", apiReposPackageFromDir))

		root.POST("/repos/:name/snapshots", ginprom.InstrumentHandlerFunc("snapshotsCreateFromRepository", apiSnapshotsCreateFromRepository))
	}

	{
		root.POST("/mirrors/:name/snapshots", ginprom.InstrumentHandlerFunc("snapshotsCreateFromMirror", apiSnapshotsCreateFromMirror))
	}

	{
		root.GET("/files", ginprom.InstrumentHandlerFunc("filesListDirs", apiFilesListDirs))
		root.POST("/files/:dir", ginprom.InstrumentHandlerFunc("filesUpload", apiFilesUpload))
		root.GET("/files/:dir", ginprom.InstrumentHandlerFunc("filesListFiles", apiFilesListFiles))
		root.DELETE("/files/:dir", ginprom.InstrumentHandlerFunc("filesDeleteDir", apiFilesDeleteDir))
		root.DELETE("/files/:dir/:name", ginprom.InstrumentHandlerFunc("filesDeleteFile", apiFilesDeleteFile))
	}

	{
		root.GET("/publish", ginprom.InstrumentHandlerFunc("publishList", apiPublishList))
		root.POST("/publish", ginprom.InstrumentHandlerFunc("publishRepoOrSnapshot", apiPublishRepoOrSnapshot))
		root.POST("/publish/:prefix", ginprom.InstrumentHandlerFunc("publishRepoOrSnapshot", apiPublishRepoOrSnapshot))
		root.PUT("/publish/:prefix/:distribution", ginprom.InstrumentHandlerFunc("publishUpdateSwitch", apiPublishUpdateSwitch))
		root.DELETE("/publish/:prefix/:distribution", ginprom.InstrumentHandlerFunc("publishDrop", apiPublishDrop))
	}

	{
		root.GET("/snapshots", ginprom.InstrumentHandlerFunc("snapshotsList", apiSnapshotsList))
		root.POST("/snapshots", ginprom.InstrumentHandlerFunc("snapshotsCreate", apiSnapshotsCreate))
		root.PUT("/snapshots/:name", ginprom.InstrumentHandlerFunc("snapshotsUpdate", apiSnapshotsUpdate))
		root.GET("/snapshots/:name", ginprom.InstrumentHandlerFunc("snapshotsShow", apiSnapshotsShow))
		root.GET("/snapshots/:name/packages", ginprom.InstrumentHandlerFunc("snapshotsSearchPackages", apiSnapshotsSearchPackages))
		root.DELETE("/snapshots/:name", ginprom.InstrumentHandlerFunc("snapshotsDrop", apiSnapshotsDrop))
		root.GET("/snapshots/:name/diff/:withSnapshot", ginprom.InstrumentHandlerFunc("snapshotsDiff", apiSnapshotsDiff))
	}

	{
		root.GET("/packages/:key", ginprom.InstrumentHandlerFunc("packagesShow", apiPackagesShow))
	}

	{
		root.GET("/graph.:ext", ginprom.InstrumentHandlerFunc("graph", apiGraph))
	}

	return router
}
