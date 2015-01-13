// Package api provides implementation of aptly REST API
package api

import (
	"github.com/gin-gonic/gin"
	"github.com/smira/aptly/aptly"
)

// Lock order acquisition (canonical):
//  1. RemoteRepoCollection
//  2. LocalRepoCollection
//  3. SnapshotCollection
//  4. PublishedRepoCollection

// GET /api/version
func apiVersion(c *gin.Context) {
	c.JSON(200, gin.H{"Version": aptly.Version})
}
