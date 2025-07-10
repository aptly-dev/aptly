package api

import (
	"github.com/gin-gonic/gin"
)

// @Summary S3 buckets
// @Description **Get list of S3 buckets**
// @Description
// @Description List configured S3 buckets.
// @Tags Status
// @Produce json
// @Success 200 {array} string "List of S3 buckets"
// @Router /api/s3 [get]
func apiS3List(c *gin.Context) {
	keys := []string{}
	// Use safe accessor to get a copy of the map
	s3Roots := context.Config().GetS3PublishRoots()
	for k := range s3Roots {
		keys = append(keys, k)
	}
	c.JSON(200, keys)
}
