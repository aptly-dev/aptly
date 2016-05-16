package api

import (
	"github.com/gin-gonic/gin"
	ctx "github.com/smira/aptly/context"
	"net/http"
	"strconv"
	"strings"
	"os/exec"
	"encoding/json"
)

var context *ctx.AptlyContext

// middleware to track API calls and call a hook script
func ApiHooks(cmd string, conf string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		var env []string
		env = append(env,
			("APTLY_METHOD="      + c.Request.Method),
			("APTLY_REQ_URL="     + c.Request.URL.String()),
			("APTLY_REMOTE_ADDR=" + c.Request.RemoteAddr),
			("APTLY_STATUS="      + strconv.Itoa(c.Writer.Status())),
			("APTLY_CONFIG="      + conf),
		)

		// collect all named params
		for _, p := range c.Params {
			env = append(env, "APTLY_PARAM_" + p.Key + "=" + p.Value)
		}

		// collect the query params
		for k, v := range c.Request.URL.Query() {
			// separate multiple query params of the same name with some "special" char
			env = append(env, "APTLY_QUERY_" + k + "=" + strings.Join(v, ","))
		}

		cmd := exec.Command(cmd)
		cmd.Env = env

		// fire and forget
		go func(){
			cmd.Run()
		}()
	}
}

// Router returns prebuilt with routes http.Handler
func Router(c *ctx.AptlyContext) http.Handler {
	context = c

	router := gin.Default()
	router.Use(gin.ErrorLogger())

	conf := context.Config()
	api_hook_cmd := conf.APIHookCmd
	if api_hook_cmd != "" {
		conf_str, _ := json.Marshal(conf)
		router.Use(ApiHooks(api_hook_cmd, string(conf_str)))
	}

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

	root := router.Group("/api")

	{
		root.GET("/version", apiVersion)
	}

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

		root.POST("/repos/:name/snapshots", apiSnapshotsCreateFromRepository)
	}

	{
		root.POST("/mirrors/:name/snapshots", apiSnapshotsCreateFromMirror)
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

	return router
}
