package api

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"os"
	"os/exec"

	"github.com/gin-gonic/gin"
	"github.com/smira/aptly/deb"
)

// GET /api/graph.:ext?layout=[vertical|horizontal(default)]
func apiGraph(c *gin.Context) {
	var (
		err    error
		output []byte
	)

	ext := c.Params.ByName("ext")
	layout := c.Request.URL.Query().Get("layout")

	factory := context.CollectionFactory()

	factory.RemoteRepoCollection().RLock()
	defer factory.RemoteRepoCollection().RUnlock()
	factory.LocalRepoCollection().RLock()
	defer factory.LocalRepoCollection().RUnlock()
	factory.SnapshotCollection().RLock()
	defer factory.SnapshotCollection().RUnlock()
	factory.PublishedRepoCollection().RLock()
	defer factory.PublishedRepoCollection().RUnlock()

	graph, err := deb.BuildGraph(factory, layout)
	if err != nil {
		c.JSON(500, err)
		return
	}

	buf := bytes.NewBufferString(graph.String())

	if ext == "dot" || ext == "gv" {
		// If the raw dot data is requested, return it as string.
		// This allows client-side rendering rather than server-side.
		c.String(200, buf.String())
		return
	}

	command := exec.Command("dot", "-T"+ext)
	command.Stderr = os.Stderr

	stdin, err := command.StdinPipe()
	if err != nil {
		c.Fail(500, err)
		return
	}

	_, err = io.Copy(stdin, buf)
	if err != nil {
		c.Fail(500, err)
		return
	}

	err = stdin.Close()
	if err != nil {
		c.Fail(500, err)
		return
	}

	output, err = command.Output()
	if err != nil {
		c.Fail(500, fmt.Errorf("unable to execute dot: %s (is graphviz package installed?)", err))
		return
	}

	mimeType := mime.TypeByExtension("." + ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	c.Data(200, mimeType, output)
}
