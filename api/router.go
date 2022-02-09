package api

import (
	"fmt"
	"log"
	"net/http"

	ctx "github.com/aptly-dev/aptly/context"
	"github.com/gin-gonic/gin"
	"github.com/nu7hatch/gouuid"
	//	"github.com/vodolaz095/ldap4gin"
)

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
		dbRequests = make(chan dbRequest)

		go acquireDatabase()

		router.Use(func(c *gin.Context) {
			var err error

			errCh := make(chan error)
			dbRequests <- dbRequest{acquiredb, errCh}

			err = <-errCh
			if err != nil {
				c.AbortWithError(500, err)
				return
			}

			defer func() {
				dbRequests <- dbRequest{releasedb, errCh}
				err = <-errCh
				if err != nil {
					c.AbortWithError(500, err)
				}
			}()

			c.Next()
		})
	}

	// prep our config fetcher ahead of need
	config := context.Config()

	router.GET("/version", apiVersion)

	router.POST("/login", func(c *gin.Context) {
		if config.UseAuth {
			log.Printf("UseAuth is enabled\n")
			username := c.PostForm("username")
			password := c.PostForm("password")
			err := Authorize(username, password)
			if err != nil {
				c.AbortWithError(403, err)
			}
		}
		token, err := uuid.NewV4()
		if err != nil {
			c.AbortWithError(500, err)
		}
		c.SetCookie("authenticatorforaptly", token.String(), 3600, "llnw.net", c.ClientIP(), true, true)
		c.String(200, "Authorized!")
	})

	authorize := router.Group("/api", func(c *gin.Context) {
		if config.UseAuth {
			_, err := c.Cookie("authenticatorforaptly")
			if err != nil {
				c.AbortWithError(403, fmt.Errorf("unauthorized"))
			} else {
				token, err := uuid.NewV4()
				if err != nil {
					c.AbortWithError(500, err)
				}
				c.SetCookie("authenticatorforaptly", token.String(), 3600, "llnw.net", c.ClientIP(), true, true)
			}
		}
	})
	{
		authorize.GET("/repos", apiReposList)
		authorize.POST("/repos", apiReposCreate)
		authorize.GET("/repos/:name", apiReposShow)
		authorize.PUT("/repos/:name", apiReposEdit)
		authorize.DELETE("/repos/:name", apiReposDrop)

		authorize.GET("/repos/:name/packages", apiReposPackagesShow)
		authorize.POST("/repos/:name/packages", apiReposPackagesAdd)
		authorize.DELETE("/repos/:name/packages", apiReposPackagesDelete)

		authorize.POST("/repos/:name/file/:dir/:file", apiReposPackageFromFile)
		authorize.POST("/repos/:name/file/:dir", apiReposPackageFromDir)

		authorize.POST("/repos/:name/include/:dir/:file", apiReposIncludePackageFromFile)
		authorize.POST("/repos/:name/include/:dir", apiReposIncludePackageFromDir)

		authorize.POST("/repos/:name/snapshots", apiSnapshotsCreateFromRepository)

		authorize.POST("/mirrors/:name/snapshots", apiSnapshotsCreateFromMirror)

		authorize.GET("/mirrors", apiMirrorsList)
		authorize.GET("/mirrors/:name", apiMirrorsShow)
		authorize.GET("/mirrors/:name/packages", apiMirrorsPackages)
		authorize.POST("/mirrors", apiMirrorsCreate)
		authorize.PUT("/mirrors/:name", apiMirrorsUpdate)
		authorize.DELETE("/mirrors/:name", apiMirrorsDrop)

		authorize.POST("/gpg/key", apiGPGAddKey)
		authorize.GET("/files", apiFilesListDirs)
		authorize.POST("/files/:dir", apiFilesUpload)
		authorize.GET("/files/:dir", apiFilesListFiles)
		authorize.DELETE("/files/:dir", apiFilesDeleteDir)
		authorize.DELETE("/files/:dir/:name", apiFilesDeleteFile)

		authorize.GET("/publish", apiPublishList)
		authorize.POST("/publish", apiPublishRepoOrSnapshot)
		authorize.POST("/publish/:prefix", apiPublishRepoOrSnapshot)
		authorize.PUT("/publish/:prefix/:distribution", apiPublishUpdateSwitch)
		authorize.DELETE("/publish/:prefix/:distribution", apiPublishDrop)

		authorize.GET("/snapshots", apiSnapshotsList)
		authorize.POST("/snapshots", apiSnapshotsCreate)
		authorize.PUT("/snapshots/:name", apiSnapshotsUpdate)
		authorize.GET("/snapshots/:name", apiSnapshotsShow)
		authorize.GET("/snapshots/:name/packages", apiSnapshotsSearchPackages)
		authorize.DELETE("/snapshots/:name", apiSnapshotsDrop)
		authorize.GET("/snapshots/:name/diff/:withSnapshot", apiSnapshotsDiff)

		authorize.GET("/packages/:key", apiPackagesShow)

		authorize.GET("/graph.:ext", apiGraph)

		authorize.POST("/db/cleanup", apiDbCleanup)

		authorize.GET("/tasks", apiTasksList)
		authorize.POST("/tasks-clear", apiTasksClear)
		authorize.GET("/tasks-wait", apiTasksWait)
		authorize.GET("/tasks/:id/wait", apiTasksWaitForTaskByID)
		authorize.GET("/tasks/:id/output", apiTasksOutputShow)
		authorize.GET("/tasks/:id/detail", apiTasksDetailShow)
		authorize.GET("/tasks/:id/return_value", apiTasksReturnValueShow)
		authorize.GET("/tasks/:id", apiTasksShow)
		authorize.DELETE("/tasks/:id", apiTasksDelete)
		authorize.POST("/tasks-dummy", apiTasksDummy)
	}

	return router
}
