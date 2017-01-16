package cmd

import (
	"bytes"
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/utils"
	"github.com/smira/commander"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

func aptlyGraph(cmd *commander.Command, args []string) error {
	var err error

	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	layout := context.Flags().Lookup("layout").Value.String()

	fmt.Printf("Generating graph...\n")
	graph, err := deb.BuildGraph(context.CollectionFactory(), layout)

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

	format := context.Flags().Lookup("format").Value.String()
	output := context.Flags().Lookup("output").Value.String()

	if filepath.Ext(output) != "" {
		format = filepath.Ext(output)[1:]
	}

	tempfilename := tempfile.Name() + "." + format

	command := exec.Command("dot", "-T"+format, "-o"+tempfilename)
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

	if output != "" {
		err = utils.CopyFile(tempfilename, output)
		if err != nil {
			return fmt.Errorf("unable to copy %s -> %s: %s", tempfilename, output, err)
		}
		_ = os.Remove(tempfilename)

		fmt.Printf("Output saved to %s\n", output)
	} else {
		fmt.Printf("Rendered to %s file: %s, trying to open it...\n", format, tempfilename)

		_ = exec.Command("open", tempfilename).Run()
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

	cmd.Flag.String("format", "png", "render graph to specified format (png, svg, pdf, etc.)")
	cmd.Flag.String("output", "", "specify output filename, default is to open result in viewer")
	cmd.Flag.String("layout", "horizontal", "create a more 'vertical' or a more 'horizontal' graph layout")

	return cmd
}
