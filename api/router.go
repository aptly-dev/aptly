package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	ctx "github.com/aptly-dev/aptly/context"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/nu7hatch/gouuid"
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

	root := router.Group("/api")

	{
		root.GET("/version", apiVersion)
	}

	// set up cookies and sessions
	token, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	store := cookie.NewStore([]byte(token.String()))
	router.Use(sessions.Sessions(token.String(), store))
	// prep our config fetcher ahead of need
	config := context.Config()

	// prep a logfile if we've set one
	if config.LogFile != "" {
		file, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		log.SetOutput(file)
	}

	router.GET("/version", apiVersion)

	var username string
	var password string
	router.POST("/login", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Options(sessions.Options{MaxAge: 30})
		if config.UseAuth {
			log.Printf("UseAuth is enabled\n")
			username = c.PostForm("username")
			password = c.PostForm("password")
			if !Authorize(username, password) {
				c.AbortWithError(403, fmt.Errorf("Authorization Failure"))
			}
			log.Printf("%s authorized from %s\n", username, c.ClientIP())
		}
		session.Set(token.String(), time.Now().Unix())
		session.Save()
		getGroups(c, username)
		c.String(200, "Authorized!")
	})

	router.POST("/logout", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Options(sessions.Options{MaxAge: -1})
		session.Save()
		c.String(200, "Deauthorized")
	})

	authorize := router.Group("/api", func(c *gin.Context) {
		session := sessions.Default(c)
		if config.UseAuth {
			if session.Get(token.String()) == nil {
				c.AbortWithError(403, fmt.Errorf("not authorized"))
			}
			session.Options(sessions.Options{MaxAge: 30})
			session.Set(token.String(), time.Now().Unix())
			session.Save()
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
