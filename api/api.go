// Package api provides implementation of aptly REST API
package api

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/query"
	"github.com/aptly-dev/aptly/task"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
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

// GET /api/ready
func apiReady(isReady *atomic.Value) func(*gin.Context) {
	return func(c *gin.Context) {
		if isReady == nil || !isReady.Load().(bool) {
			c.JSON(503, gin.H{"Status": "Aptly is unavailable"})
			return
		}

		c.JSON(200, gin.H{"Status": "Aptly is ready"})
	}
}

// GET /api/healthy
func apiHealthy(c *gin.Context) {
	c.JSON(200, gin.H{"Status": "Aptly is healthy"})
}

type dbRequestKind int

const (
	acquiredb dbRequestKind = iota
	releasedb
)

type dbRequest struct {
	kind dbRequestKind
	err  chan<- error
}

var dbRequests chan dbRequest

// Acquire database lock and release it when not needed anymore.
//
// Should be run in a goroutine!
func acquireDatabase() {
	clients := 0
	for request := range dbRequests {
		var err error

		switch request.kind {
		case acquiredb:
			if clients == 0 {
				err = context.ReOpenDatabase()
			}

			request.err <- err

			if err == nil {
				clients++
			}
		case releasedb:
			clients--
			if clients == 0 {
				err = context.CloseDatabase()
			} else {
				err = nil
			}

			request.err <- err
		}
	}
}

// Should be called before database access is needed in any api call.
// Happens per default for each api call. It is important that you run
// runTaskInBackground to run a task which accquire database.
// Important do not forget to defer to releaseDatabaseConnection
func acquireDatabaseConnection() error {
	if dbRequests == nil {
		return nil
	}

	errCh := make(chan error)
	dbRequests <- dbRequest{acquiredb, errCh}

	return <-errCh
}

// Release database connection when not needed anymore
func releaseDatabaseConnection() error {
	if dbRequests == nil {
		return nil
	}

	errCh := make(chan error)
	dbRequests <- dbRequest{releasedb, errCh}
	return <-errCh
}

// runs tasks in background. Acquires database connection first.
func runTaskInBackground(name string, resources []string, proc task.Process) (task.Task, *task.ResourceConflictError) {
	return context.TaskList().RunTaskInBackground(name, resources, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		err := acquireDatabaseConnection()

		if err != nil {
			return nil, err
		}

		defer releaseDatabaseConnection()
		return proc(out, detail)
	})
}

func truthy(value interface{}) bool {
	if value == nil {
		return false
	}
	switch value.(type) {
	case string:
		switch strings.ToLower(value.(string)) {
		case "n", "no", "f", "false", "0", "off":
			return false
		default:
			return true
		}
	case int:
		return !(value.(int) == 0)
	case bool:
		return value.(bool)
	}
	return true
}

func maybeRunTaskInBackground(c *gin.Context, name string, resources []string, proc task.Process) {
	// Run this task in background if configured globally or per-request
	background := truthy(c.DefaultQuery("_async", strconv.FormatBool(context.Config().AsyncAPI)))
	if background {
		log.Debug().Msg("Executing task asynchronously")
		task, conflictErr := runTaskInBackground(name, resources, proc)
		if conflictErr != nil {
			AbortWithJSONError(c, 409, conflictErr)
			return
		}
		c.JSON(202, task)
	} else {
		log.Debug().Msg("Executing task synchronously")
		task, conflictErr := runTaskInBackground(name, resources, proc)
		if conflictErr != nil {
			AbortWithJSONError(c, 409, conflictErr)
			return
		}

		// wait for task to finish
		context.TaskList().WaitForTaskByID(task.ID)

		retValue, _ := context.TaskList().GetTaskReturnValueByID(task.ID)
		err, _ := context.TaskList().GetTaskErrorByID(task.ID)
		context.TaskList().DeleteTaskByID(task.ID)
		if err != nil {
			AbortWithJSONError(c, retValue.Code, err)
			return
		}
		if retValue != nil {
			c.JSON(retValue.Code, retValue.Value)
		} else {
			c.JSON(http.StatusOK, nil)
		}
	}
}

// Common piece of code to show list of packages,
// with searching & details if requested
func showPackages(c *gin.Context, reflist *deb.PackageRefList, collectionFactory *deb.CollectionFactory) {
	result := []*deb.Package{}

	list, err := deb.NewPackageListFromRefList(reflist, collectionFactory.PackageCollection(), nil)
	if err != nil {
		AbortWithJSONError(c, 404, err)
		return
	}

	queryS := c.Request.URL.Query().Get("q")
	if queryS != "" {
		q, err := query.Parse(c.Request.URL.Query().Get("q"))
		if err != nil {
			AbortWithJSONError(c, 400, err)
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
				AbortWithJSONError(c, 400, fmt.Errorf("unable to determine list of architectures, please specify explicitly"))
				return
			}
		}

		list.PrepareIndex()

		list, err = list.Filter([]deb.PackageQuery{q}, withDeps,
			nil, context.DependencyOptions(), architecturesList)
		if err != nil {
			AbortWithJSONError(c, 500, fmt.Errorf("unable to search: %s", err))
			return
		}
	}

	// filter packages by version
	if c.Request.URL.Query().Get("maximumVersion") == "1" {
		list.PrepareIndex()
		list.ForEach(func(p *deb.Package) error {
			versionQ, err := query.Parse(fmt.Sprintf("Name (%s), $Version (<= %s)", p.Name, p.Version))
			if err != nil {
				fmt.Println("filter packages by version, query string parse err: ", err)
				c.AbortWithError(500, fmt.Errorf("unable to parse %s maximum version query string: %s", p.Name, err))
			} else {
				tmpList, err := list.Filter([]deb.PackageQuery{versionQ}, false,
					nil, 0, []string{})

				if err == nil {
					if tmpList.Len() > 0 {
						tmpList.ForEach(func(tp *deb.Package) error {
							list.Remove(tp)
							return nil
						})
						list.Add(p)
					}
				} else {
					fmt.Println("filter packages by version, filter err: ", err)
					c.AbortWithError(500, fmt.Errorf("unable to get %s maximum version: %s", p.Name, err))
				}
			}

			return nil
		})
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

func AbortWithJSONError(c *gin.Context, code int, err error) *gin.Error {
	c.Writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	return c.AbortWithError(code, err)
}
