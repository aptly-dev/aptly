package deb

import (
	"fmt"
	"github.com/awalterschulze/gographviz"
	"strings"
)

// BuildGraph generates graph contents from aptly object database
func BuildGraph(collectionFactory *CollectionFactory) (gographviz.Interface, error) {
	var err error

	graph := gographviz.NewEscape()
	graph.SetDir(true)
	graph.SetName("aptly")

	existingNodes := map[string]bool{}

	err = collectionFactory.RemoteRepoCollection().ForEach(func(repo *RemoteRepo) error {
		err := collectionFactory.RemoteRepoCollection().LoadComplete(repo)
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

	err = collectionFactory.LocalRepoCollection().ForEach(func(repo *LocalRepo) error {
		err := collectionFactory.LocalRepoCollection().LoadComplete(repo)
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

	collectionFactory.SnapshotCollection().ForEach(func(snapshot *Snapshot) error {
		existingNodes[snapshot.UUID] = true
		return nil
	})

	err = collectionFactory.SnapshotCollection().ForEach(func(snapshot *Snapshot) error {
		err := collectionFactory.SnapshotCollection().LoadComplete(snapshot)
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

	collectionFactory.PublishedRepoCollection().ForEach(func(repo *PublishedRepo) error {
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
