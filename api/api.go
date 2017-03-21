// Package api provides implementation of aptly REST API
package api

import (
	"fmt"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/query"
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

const (
	acquiredb = iota
	releasedb
)

// Flushes all collections which cache in-memory objects
func flushColections() {
	// lock everything to eliminate in-progress calls
	r := context.CollectionFactory().RemoteRepoCollection()
	r.Lock()
	defer r.Unlock()

	l := context.CollectionFactory().LocalRepoCollection()
	l.Lock()
	defer l.Unlock()

	s := context.CollectionFactory().SnapshotCollection()
	s.Lock()
	defer s.Unlock()

	p := context.CollectionFactory().PublishedRepoCollection()
	p.Lock()
	defer p.Unlock()

	// all collections locked, flush them
	context.CollectionFactory().Flush()
}

// Periodically flushes CollectionFactory to free up memory used by
// collections, flushing caches. If the two channels are provided,
// they are used to acquire and release the database.
//
// Should be run in goroutine!
func cacheFlusher(requests chan int, acks chan error) {
	ticker := time.Tick(15 * time.Minute)

	for {
		<-ticker

		// if aptly API runs in -no-lock mode,
		// caches are flushed when DB is closed anyway, no need
		// to flush them here
		if requests == nil {
			flushColections()
		}
	}
}

// Acquire database lock and release it when not needed anymore. Two
// channels must be provided. The first one is to receive requests to
// acquire/release the database and the second one is to send acks.
//
// Should be run in a goroutine!
func acquireDatabase(requests chan int, acks chan error) {
	clients := 0
	for {
		request := <-requests
		switch request {
		case acquiredb:
			if clients == 0 {
				acks <- context.ReOpenDatabase()
			} else {
				acks <- nil
			}
			clients++
		case releasedb:
			clients--
			if clients == 0 {
				flushColections()
				acks <- context.CloseDatabase()
			} else {
				acks <- nil
			}
		}
	}
}

// Common piece of code to show list of packages,
// with searching & details if requested
func showPackages(c *gin.Context, reflist *deb.PackageRefList) {
	result := []*deb.Package{}

	list, err := deb.NewPackageListFromRefList(reflist, context.CollectionFactory().PackageCollection(), nil)
	if err != nil {
		c.Fail(404, err)
		return
	}

	queryS := c.Request.URL.Query().Get("q")
	if queryS != "" {
		q, err := query.Parse(c.Request.URL.Query().Get("q"))
		if err != nil {
			c.Fail(400, err)
			return
		}

		withDeps := c.Request.URL.Query().Get("withDeps") == "1"
		architecturesList := []string{}

		if withDeps {
			if len(context.ArchitecturesList()) > 0 {
				architecturesList = context.ArchitecturesList()
			} else {
				architecturesList = list.Architectures(false)
			}

			sort.Strings(architecturesList)

			if len(architecturesList) == 0 {
				c.Fail(400, fmt.Errorf("unable to determine list of architectures, please specify explicitly"))
				return
			}
		}

		list.PrepareIndex()

		list, err = list.Filter([]deb.PackageQuery{q}, withDeps,
			nil, context.DependencyOptions(), architecturesList)
		if err != nil {
			c.Fail(500, fmt.Errorf("unable to search: %s", err))
			return
		}
	}

	if c.Request.URL.Query().Get("format") == "details" {
		list.ForEach(func(p *deb.Package) error {
			result = append(result, p)
			return nil
		})

		c.JSON(200, result)
	} else {
		c.JSON(200, list.Strings())
	}
}
