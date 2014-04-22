package cmd

import (
	"bytes"
	"code.google.com/p/gographviz"
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
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

	existingNodes := map[string]bool{}

	fmt.Printf("Loading mirrors...\n")

	err = context.CollectionFactory().RemoteRepoCollection().ForEach(func(repo *deb.RemoteRepo) error {
		err := context.CollectionFactory().RemoteRepoCollection().LoadComplete(repo)
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
		existingNodes[repo.UUID] = true
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Loading local repos...\n")

	err = context.CollectionFactory().LocalRepoCollection().ForEach(func(repo *deb.LocalRepo) error {
		err := context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
		if err != nil {
			return err
		}

		graph.AddNode("aptly", graphvizEscape(repo.UUID), map[string]string{
			"shape":     "Mrecord",
			"style":     "filled",
			"fillcolor": "mediumseagreen",
			"label": graphvizEscape(fmt.Sprintf("{Repo %s|comment: %s|pkgs: %d}",
				repo.Name, repo.Comment, repo.NumPackages())),
		})
		existingNodes[repo.UUID] = true
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Loading snapshots...\n")

	context.CollectionFactory().SnapshotCollection().ForEach(func(snapshot *deb.Snapshot) error {
		existingNodes[snapshot.UUID] = true
		return nil
	})

	err = context.CollectionFactory().SnapshotCollection().ForEach(func(snapshot *deb.Snapshot) error {
		err := context.CollectionFactory().SnapshotCollection().LoadComplete(snapshot)
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

		if snapshot.SourceKind == "repo" || snapshot.SourceKind == "local" || snapshot.SourceKind == "snapshot" {
			for _, uuid := range snapshot.SourceIDs {
				_, exists := existingNodes[uuid]
				if exists {
					graph.AddEdge(graphvizEscape(uuid), "", graphvizEscape(snapshot.UUID), "", true, nil)
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("Loading published repos...\n")

	context.CollectionFactory().PublishedRepoCollection().ForEach(func(repo *deb.PublishedRepo) error {
		graph.AddNode("aptly", graphvizEscape(repo.UUID), map[string]string{
			"shape":     "Mrecord",
			"style":     "filled",
			"fillcolor": "darkolivegreen1",
			"label":     graphvizEscape(fmt.Sprintf("{Published %s/%s|comp: %s|arch: %s}", repo.Prefix, repo.Distribution, repo.Component, strings.Join(repo.Architectures, ", "))),
		})

		_, exists := existingNodes[repo.SourceUUID]
		if exists {
			graph.AddEdge(graphvizEscape(repo.SourceUUID), "", graphvizEscape(repo.UUID), "", true, nil)
		}

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
