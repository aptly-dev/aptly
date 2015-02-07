package api

import (
	"github.com/gin-gonic/gin"
)

// GET /api/packages/:key
func apiPackagesShow(c *gin.Context) {
	p, err := context.CollectionFactory().PackageCollection().ByKey([]byte(c.Params.ByName("key")))
	if err != nil {
		c.Fail(404, err)
		return
	}

	c.JSON(200, p)
}
