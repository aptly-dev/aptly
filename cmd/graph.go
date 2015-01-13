package cmd

import (
	"bytes"
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

func aptlyGraph(cmd *commander.Command, args []string) error {
	var err error

	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	fmt.Printf("Generating graph...\n")
	graph, err := deb.BuildGraph(context.CollectionFactory())
	if err != nil {
		return err
	}

	buf := bytes.NewBufferString(graph.String())

	tempfile, err := ioutil.TempFile("", "aptly-graph")
	if err != nil {
		return err
	}
	tempfile.Close()
	os.Remove(tempfile.Name())

	tempfilename := tempfile.Name() + ".png"

	command := exec.Command("dot", "-Tpng", "-o"+tempfilename)
	command.Stderr = os.Stderr

	stdin, err := command.StdinPipe()
	if err != nil {
		return err
	}

	err = command.Start()
	if err != nil {
		return fmt.Errorf("unable to execute dot: %s (is graphviz package installed?)", err)
	}

	_, err = io.Copy(stdin, buf)
	if err != nil {
		return err
	}

	err = stdin.Close()
	if err != nil {
		return err
	}

	err = command.Wait()
	if err != nil {
		return err
	}

	err = exec.Command("open", tempfilename).Run()
	if err != nil {
		fmt.Printf("Rendered to PNG file: %s\n", tempfilename)
		err = nil
	}

	return err
}

func makeCmdGraph() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyGraph,
		UsageLine: "graph",
		Short:     "render graph of relationships",
		Long: `
Command graph displays relationship between mirrors, local repositories,
snapshots and published repositories using graphviz package to render
graph as an image.

Example:

  $ aptly graph
`,
	}

	return cmd
}
