package api

import (
	"github.com/gin-gonic/gin"
)

// GET /api/packages/:key
func apiPackagesShow(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	p, err := collectionFactory.PackageCollection().ByKey([]byte(c.Params.ByName("key")))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	c.JSON(200, p)
}

// GET /api/packages
func apiPackages(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PackageCollection()
	showPackages(c, collection.AllPackageRefs(), collectionFactory)
}
