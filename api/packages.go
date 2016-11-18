package api

import (
	"github.com/gin-gonic/gin"
)

// GET /api/packages/:key
func apiPackagesShow(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	p, err := collectionFactory.PackageCollection().ByKey([]byte(c.Params.ByName("key")))
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	c.JSON(200, p)
}
