package main

import (
	"bytes"
	"code.google.com/p/gographviz"
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

func graphvizEscape(s string) string {
	return fmt.Sprintf("\"%s\"", strings.Replace(s, "\"", "\\\"", 0))
}

func aptlyGraph(cmd *commander.Command, args []string) error {
	var err error

	graph := gographviz.NewGraph()
	graph.SetDir(true)
	graph.SetName("aptly")

	fmt.Printf("Loading mirrors...\n")

	repoCollection := debian.NewRemoteRepoCollection(context.database)

	err = repoCollection.ForEach(func(repo *debian.RemoteRepo) error {
		err := repoCollection.LoadComplete(repo)
		if err != nil {
			return err
		}

		graph.AddNode("aptly", graphvizEscape(repo.UUID), map[string]string{
			"shape":     "Mrecord",
			"style":     "filled",
			"fillcolor": "darkgoldenrod1",
			"label": graphvizEscape(fmt.Sprintf("{Mirror %s|url: %s|dist: %s|comp: %s|arch: %s|pkgs: %d}",
				repo.Name, repo.ArchiveRoot, repo.Distribution, strings.Join(repo.Components, ", "),
				strings.Join(repo.Architectures, ", "), repo.NumPackages())),
		})
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Loading snapshots...\n")

	snapshotCollection := debian.NewSnapshotCollection(context.database)

	err = snapshotCollection.ForEach(func(snapshot *debian.Snapshot) error {
		err := snapshotCollection.LoadComplete(snapshot)
		if err != nil {
			return err
		}

		description := snapshot.Description
		if snapshot.SourceKind == "repo" {
			description = "Snapshot from repo"
		}

		graph.AddNode("aptly", graphvizEscape(snapshot.UUID), map[string]string{
			"shape":     "Mrecord",
			"style":     "filled",
			"fillcolor": "cadetblue1",
			"label":     graphvizEscape(fmt.Sprintf("{Snapshot %s|%s|pkgs: %d}", snapshot.Name, description, snapshot.NumPackages())),
		})

		if snapshot.SourceKind == "repo" {
			for _, uuid := range snapshot.SourceIDs {
				graph.AddEdge(graphvizEscape(uuid), "", graphvizEscape(snapshot.UUID), "", true, nil)
			}
		} else if snapshot.SourceKind == "snapshot" {
			for _, uuid := range snapshot.SourceIDs {
				graph.AddEdge(graphvizEscape(uuid), "", graphvizEscape(snapshot.UUID), "", true, nil)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Loading published repos...\n")

	publishedCollection := debian.NewPublishedRepoCollection(context.database)

	publishedCollection.ForEach(func(repo *debian.PublishedRepo) error {
		graph.AddNode("aptly", graphvizEscape(repo.UUID), map[string]string{
			"shape":     "Mrecord",
			"style":     "filled",
			"fillcolor": "darkolivegreen1",
			"label":     graphvizEscape(fmt.Sprintf("{Published %s/%s|comp: %s|arch: %s}", repo.Prefix, repo.Distribution, repo.Component, strings.Join(repo.Architectures, ", "))),
		})

		graph.AddEdge(graphvizEscape(repo.SnapshotUUID), "", graphvizEscape(repo.UUID), "", true, nil)

		return nil
	})

	fmt.Printf("Generating graph...\n")

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
		Short:     "display graph of dependencies between aptly objects (requires graphviz)",
		Long: `
Command graph displays relationship between mirrors, snapshots and published repositories using
graphviz package to render graph as image.

ex:
  $ aptly graph
`,
		Flag: *flag.NewFlagSet("aptly-graph", flag.ExitOnError),
	}

	return cmd
}
