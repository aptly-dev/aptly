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

	output := context.Flags().Lookup("output").Value.String()

	tempfilename := tempfile.Name() + "." + output

	command := exec.Command("dot", "-T"+output, "-o"+tempfilename)
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

	fmt.Printf("Rendered to %s file: %s, trying to open it...\n", output, tempfilename)

	_ = exec.Command("open", tempfilename).Run()

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

	cmd.Flag.String("output", "png", "reder graph to output kind (png, svg, pdf, etc.)")

	return cmd
}
