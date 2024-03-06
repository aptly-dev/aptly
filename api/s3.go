package api

import (
	"github.com/gin-gonic/gin"
)

// GET /api/s3
func apiS3List(c *gin.Context) {
	keys := []string{}
	for k := range context.Config().S3PublishRoots {
		keys = append(keys, k)
	}
	c.JSON(200, keys)
}
