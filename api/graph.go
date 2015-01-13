package api

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"github.com/gin-gonic/gin"
	"github.com/smira/aptly/graph"
)

// GET /api/graph
func apiGraph(c *gin.Context) {
	var (
		err      error
		output []byte
	)

	graph, err := graph.BuildGraph(context)
	if err != nil {
		c.JSON(500, err)
		return
	}

	buf := bytes.NewBufferString(graph.String())

	command := exec.Command("dot", "-Tpng")
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

	c.Data(200, "image/png", output)
}
