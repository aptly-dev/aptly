package api

import (
	"github.com/gin-gonic/gin"
)

// @Summary JFrog repositories
// @Description **Get list of JFrog repositories**
// @Description
// @Description List configured JFrog publish endpoints.
// @Tags Status
// @Produce json
// @Success 200 {array} string "List of JFrog publish endpoints"
// @Router /api/jfrog [get]
func apiJFrogList(c *gin.Context) {
	keys := []string{}
	for k := range context.Config().JFrogPublishRoots {
		keys = append(keys, k)
	}
	c.JSON(200, keys)
}
