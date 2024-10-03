package api

import (
	"fmt"
	"syscall"

	"github.com/gin-gonic/gin"
)

type diskFree struct {
	// Storage size [MiB]
	Total uint64
	// Available Storage [MiB]
	Free uint64
	// Percentage Full
	PercentFull float32
}

// @Summary Get Storage Utilization
// @Description **Get disk free information of aptly storage**
// @Tags Status
// @Produce json
// @Success 200 {object} diskFree "Storage information"
// @Failure 400 {object} Error "Internal Error"
// @Router /api/storage [get]
func apiDiskFree(c *gin.Context) {
	var df diskFree

	fs := context.Config().GetRootDir()

	var stat syscall.Statfs_t
	err := syscall.Statfs(fs, &stat)
	if err != nil {
		AbortWithJSONError(c, 400, fmt.Errorf("Error getting storage info on %s: %s", fs, err))
		return
	}

	df.Total = uint64(stat.Blocks) * uint64(stat.Bsize) / 1048576
	df.Free = uint64(stat.Bavail) * uint64(stat.Bsize) / 1048576
	df.PercentFull = 100.0 - float32(stat.Bavail)/float32(stat.Blocks)*100.0

	c.JSON(200, df)
}
