package graph

import (
	"code.google.com/p/gographviz"
	"fmt"
	"strings"
	"github.com/smira/aptly/context"
	"github.com/smira/aptly/deb"
)

func BuildGraph(context *context.AptlyContext) (gographviz.Interface, error) {
	var err error

	graph := gographviz.NewEscape()
	graph.SetDir(true)
	graph.SetName("aptly")

	existingNodes := map[string]bool{}

	err = context.CollectionFactory().RemoteRepoCollection().ForEach(func(repo *deb.RemoteRepo) error {
		err := context.CollectionFactory().RemoteRepoCollection().LoadComplete(repo)
		if err != nil {
			return err
		}

		graph.AddNode("aptly", repo.UUID, map[string]string{
			"shape":     "Mrecord",
			"style":     "filled",
			"fillcolor": "darkgoldenrod1",
			"label": fmt.Sprintf("{Mirror %s|url: %s|dist: %s|comp: %s|arch: %s|pkgs: %d}",
				repo.Name, repo.ArchiveRoot, repo.Distribution, strings.Join(repo.Components, ", "),
				strings.Join(repo.Architectures, ", "), repo.NumPackages()),
		})
		existingNodes[repo.UUID] = true
		return nil
	})

	if err != nil {
		return nil, err
	}

	err = context.CollectionFactory().LocalRepoCollection().ForEach(func(repo *deb.LocalRepo) error {
		err := context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
		if err != nil {
			return err
		}

		graph.AddNode("aptly", repo.UUID, map[string]string{
			"shape":     "Mrecord",
			"style":     "filled",
			"fillcolor": "mediumseagreen",
			"label": fmt.Sprintf("{Repo %s|comment: %s|pkgs: %d}",
				repo.Name, repo.Comment, repo.NumPackages()),
		})
		existingNodes[repo.UUID] = true
		return nil
	})

	if err != nil {
		return nil, err
	}

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

		graph.AddNode("aptly", snapshot.UUID, map[string]string{
			"shape":     "Mrecord",
			"style":     "filled",
			"fillcolor": "cadetblue1",
			"label":     fmt.Sprintf("{Snapshot %s|%s|pkgs: %d}", snapshot.Name, description, snapshot.NumPackages()),
		})

		if snapshot.SourceKind == "repo" || snapshot.SourceKind == "local" || snapshot.SourceKind == "snapshot" {
			for _, uuid := range snapshot.SourceIDs {
				_, exists := existingNodes[uuid]
				if exists {
					graph.AddEdge(uuid, snapshot.UUID, true, nil)
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	context.CollectionFactory().PublishedRepoCollection().ForEach(func(repo *deb.PublishedRepo) error {
		graph.AddNode("aptly", repo.UUID, map[string]string{
			"shape":     "Mrecord",
			"style":     "filled",
			"fillcolor": "darkolivegreen1",
			"label": fmt.Sprintf("{Published %s/%s|comp: %s|arch: %s}", repo.Prefix, repo.Distribution,
				strings.Join(repo.Components(), " "), strings.Join(repo.Architectures, ", ")),
		})

		for _, uuid := range repo.Sources {
			_, exists := existingNodes[uuid]
			if exists {
				graph.AddEdge(uuid, repo.UUID, true, nil)
			}
		}

		return nil
	})

	return graph, nil
}