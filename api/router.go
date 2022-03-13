package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/aptly-dev/aptly/aptly"
	ctx "github.com/aptly-dev/aptly/context"
	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	_ "github.com/aptly-dev/aptly/docs" // import docs
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	uuid "github.com/nu7hatch/gouuid"
)

var context *ctx.AptlyContext

func apiMetricsGet() gin.HandlerFunc {
	return func(c *gin.Context) {
		countPackagesByRepos()
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}
}

func redirectSwagger(c *gin.Context) {
	if c.Request.URL.Path == "/docs/" {
		c.Redirect(http.StatusMovedPermanently, "/docs/index.html")
		return
	}
	c.Next()
}

// Router returns prebuilt with routes http.Handler
// @title           Aptly API
// @version         1.0
// @description     Aptly REST API Documentation

// @contact.name   Aptly
// @contact.url    http://github.com/aptly-dev/aptly
// @contact.email  support@aptly.info

// @license.name  MIT License
// @license.url   http://www.

// @BasePath  /api
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

	if c.Config().EnableSwaggerEndpoint {
		router.Use(redirectSwagger)
		url := ginSwagger.URL("/docs/doc.json")
		router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
	}

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
		api.GET("/storage", apiDiskFree)

		isReady := &atomic.Value{}
		isReady.Store(false)
		defer isReady.Store(true)
		api.GET("/ready", apiReady(isReady))
		api.GET("/healthy", apiHealthy)
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
		authorize.POST("/repos/:name/copy/:src/:file", apiReposCopyPackage)

		authorize.POST("/repos/:name/include/:dir/:file", apiReposIncludePackageFromFile)
		authorize.POST("/repos/:name/include/:dir", apiReposIncludePackageFromDir)

		authorize.POST("/repos/:name/snapshots", apiSnapshotsCreateFromRepository)
	}

	{
		authorize.POST("/mirrors/:name/snapshots", apiSnapshotsCreateFromMirror)
	}

	{
		authorize.GET("/mirrors", apiMirrorsList)
		authorize.GET("/mirrors/:name", apiMirrorsShow)
		authorize.GET("/mirrors/:name/packages", apiMirrorsPackages)
		authorize.POST("/mirrors", apiMirrorsCreate)
		authorize.PUT("/mirrors/:name", apiMirrorsUpdate)
		authorize.DELETE("/mirrors/:name", apiMirrorsDrop)
	}

	{
		authorize.POST("/gpg/key", apiGPGAddKey)
	}

	{
		authorize.GET("/s3", apiS3List)
	}

	{
		authorize.GET("/files", apiFilesListDirs)
		authorize.POST("/files/:dir", apiFilesUpload)
		authorize.GET("/files/:dir", apiFilesListFiles)
		authorize.DELETE("/files/:dir", apiFilesDeleteDir)
		authorize.DELETE("/files/:dir/:name", apiFilesDeleteFile)
	}

	{
		authorize.GET("/publish", apiPublishList)
		authorize.POST("/publish", apiPublishRepoOrSnapshot)
		authorize.POST("/publish/:prefix", apiPublishRepoOrSnapshot)
		authorize.PUT("/publish/:prefix/:distribution", apiPublishUpdateSwitch)
		authorize.DELETE("/publish/:prefix/:distribution", apiPublishDrop)
	}

	{
		authorize.GET("/snapshots", apiSnapshotsList)
		authorize.POST("/snapshots", apiSnapshotsCreate)
		authorize.PUT("/snapshots/:name", apiSnapshotsUpdate)
		authorize.GET("/snapshots/:name", apiSnapshotsShow)
		authorize.GET("/snapshots/:name/packages", apiSnapshotsSearchPackages)
		authorize.DELETE("/snapshots/:name", apiSnapshotsDrop)
		authorize.GET("/snapshots/:name/diff/:withSnapshot", apiSnapshotsDiff)
		authorize.POST("/snapshots/:name/merge", apiSnapshotsMerge)
		authorize.POST("/snapshots/:name/pull", apiSnapshotsPull)
	}

	{
		authorize.GET("/packages/:key", apiPackagesShow)
		authorize.GET("/packages", apiPackages)
	}

	{
		authorize.GET("/graph.:ext", apiGraph)
	}
	{
		authorize.POST("/db/cleanup", apiDbCleanup)
	}
	{
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
