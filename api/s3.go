package api

import (
	"github.com/gin-gonic/gin"
)

// @Summary Get S3 buckets
// @Description Get list of S3 buckets.
// @Tags S3
// @Produce  json
// @Success 200 {array} string "List of S3 buckets"
// @Router /api/s3 [get]
func apiS3List(c *gin.Context) {
	keys := []string{}
	for k := range context.Config().S3PublishRoots {
		keys = append(keys, k)
	}
	c.JSON(200, keys)
}
