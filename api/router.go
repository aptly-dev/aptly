package api

import (
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

	// prep a logfile if we've set one
	if config.LogFile != "" {
		file, err := os.OpenFile(config.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		log.SetOutput(file)
	}

	// set up cookies and sessions
	token, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}

	store := cookie.NewStore([]byte(token.String()))
	router.Use(sessions.Sessions(token.String(), store))

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
		root.GET("/repos", apiReposList)
		root.POST("/repos", apiReposCreate)
		root.GET("/repos/:name", apiReposShow)
		root.PUT("/repos/:name", apiReposEdit)
		root.DELETE("/repos/:name", apiReposDrop)

		root.GET("/repos/:name/packages", apiReposPackagesShow)
		root.POST("/repos/:name/packages", apiReposPackagesAdd)
		root.DELETE("/repos/:name/packages", apiReposPackagesDelete)

		root.POST("/repos/:name/file/:dir/:file", apiReposPackageFromFile)
		root.POST("/repos/:name/file/:dir", apiReposPackageFromDir)

		root.POST("/repos/:name/include/:dir/:file", apiReposIncludePackageFromFile)
		root.POST("/repos/:name/include/:dir", apiReposIncludePackageFromDir)

		root.POST("/repos/:name/snapshots", apiSnapshotsCreateFromRepository)
	}

	{
		root.POST("/mirrors/:name/snapshots", apiSnapshotsCreateFromMirror)
	}

	{
		root.GET("/mirrors", apiMirrorsList)
		root.GET("/mirrors/:name", apiMirrorsShow)
		root.GET("/mirrors/:name/packages", apiMirrorsPackages)
		root.POST("/mirrors", apiMirrorsCreate)
		root.PUT("/mirrors/:name", apiMirrorsUpdate)
		root.DELETE("/mirrors/:name", apiMirrorsDrop)
	}

	{
		root.POST("/gpg/key", apiGPGAddKey)
	}

	{
		root.GET("/files", apiFilesListDirs)
		root.POST("/files/:dir", apiFilesUpload)
		root.GET("/files/:dir", apiFilesListFiles)
		root.DELETE("/files/:dir", apiFilesDeleteDir)
		root.DELETE("/files/:dir/:name", apiFilesDeleteFile)
	}

	{
		root.GET("/publish", apiPublishList)
		root.POST("/publish", apiPublishRepoOrSnapshot)
		root.POST("/publish/:prefix", apiPublishRepoOrSnapshot)
		root.PUT("/publish/:prefix/:distribution", apiPublishUpdateSwitch)
		root.DELETE("/publish/:prefix/:distribution", apiPublishDrop)
	}

	{
		root.GET("/snapshots", apiSnapshotsList)
		root.POST("/snapshots", apiSnapshotsCreate)
		root.PUT("/snapshots/:name", apiSnapshotsUpdate)
		root.GET("/snapshots/:name", apiSnapshotsShow)
		root.GET("/snapshots/:name/packages", apiSnapshotsSearchPackages)
		root.DELETE("/snapshots/:name", apiSnapshotsDrop)
		root.GET("/snapshots/:name/diff/:withSnapshot", apiSnapshotsDiff)
	}

	{
		root.GET("/packages/:key", apiPackagesShow)
	}

	{
		root.GET("/graph.:ext", apiGraph)
	}
	{
		root.POST("/db/cleanup", apiDbCleanup)
	}
	{
		root.GET("/tasks", apiTasksList)
		root.POST("/tasks-clear", apiTasksClear)
		root.GET("/tasks-wait", apiTasksWait)
		root.GET("/tasks/:id/wait", apiTasksWaitForTaskByID)
		root.GET("/tasks/:id/output", apiTasksOutputShow)
		root.GET("/tasks/:id/detail", apiTasksDetailShow)
		root.GET("/tasks/:id/return_value", apiTasksReturnValueShow)
		root.GET("/tasks/:id", apiTasksShow)
		root.DELETE("/tasks/:id", apiTasksDelete)
		root.POST("/tasks-dummy", apiTasksDummy)
	}

	return router
}
