package api

import (
	"github.com/gin-gonic/gin"
	ctx "github.com/smira/aptly/context"
	"net/http"
)

var context *ctx.AptlyContext

// Router returns prebuilt with routes http.Handler
func Router(c *ctx.AptlyContext) http.Handler {
	context = c

	router := gin.Default()
	router.Use(gin.ErrorLogger())

	root := router.Group("/api")
	{
		root.GET("/repos", apiReposList)
		root.POST("/repos", apiReposCreate)
		root.GET("/repos/:name", apiReposShow)
		root.PUT("/repos/:name", apiReposEdit)
		root.DELETE("/repos/:name", apiReposDrop)

		root.GET("/repos/:name/packages", apiReposPackagesShow)
		root.POST("/repos/:name/packages", apiReposPackagesAdd)
	}

	return router
}
