package api

import (
	_ "github.com/aptly-dev/aptly/deb" // for swagger
	"github.com/gin-gonic/gin"
)

// @Summary Show packages
// @Description **Show information about package by package key**
// @Description Package keys could be obtained from various GET .../packages APIs.
// @Tags Packages
// @Produce json
// @Param key path string true "package key (unique package identifier)"
// @Success 200 {object} deb.Package "OK"
// @Failure 404 {object} Error "Not Found"
// @Router /api/packages/{key} [get]
func apiPackagesShow(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	p, err := collectionFactory.PackageCollection().ByKey([]byte(c.Params.ByName("key")))
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	c.JSON(200, p)
}

// @Summary Get packages
// @Description Get list of packages.
// @Tags Packages
// @Consume  json
// @Produce  json
// @Param q query string false "search query"
// @Param format query string false "format: `details` for more detailed information"
// @Success 200 {array} string "List of packages"
// @Router /api/packages [get]
func apiPackages(c *gin.Context) {
	collectionFactory := context.NewCollectionFactory()
	collection := collectionFactory.PackageCollection()
	showPackages(c, collection.AllPackageRefs(), collectionFactory)
}
