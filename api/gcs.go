package api

import (
	"github.com/gin-gonic/gin"
)

// @Summary GCS buckets
// @Description **Get list of GCS buckets**
// @Description
// @Description List configured GCS buckets.
// @Tags Status
// @Produce json
// @Success 200 {array} string "List of GCS buckets"
// @Router /api/gcs [get]
func apiGCSList(c *gin.Context) {
	keys := []string{}
	for k := range context.Config().GCSPublishRoots {
		keys = append(keys, k)
	}
	c.JSON(200, keys)
}
