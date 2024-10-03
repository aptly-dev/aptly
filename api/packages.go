package api

import (
	"github.com/gin-gonic/gin"
)

// @Summary TODO
// @Description **ToDo**
// @Description To Do
// @Tags Packages
// @Produce json
// @Success 200 {object} string "msg"
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
