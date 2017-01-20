package deb

import (
	"fmt"
	"github.com/awalterschulze/gographviz"
	"strings"
)

// BuildGraph generates graph contents from aptly object database
func BuildGraph(collectionFactory *CollectionFactory, layout string) (gographviz.Interface, error) {
	var err error

	graph := gographviz.NewEscape()
	graph.SetDir(true)
	graph.SetName("aptly")

	var labelStart string
	var labelEnd string

	switch layout {
		case "vertical":
			graph.AddAttr("aptly", "rankdir", "LR")
			labelStart = ""
			labelEnd = ""
		case "horizontal":
			fallthrough
		default:
			labelStart = "{"
			labelEnd = "}"
	}

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
			"label": fmt.Sprintf("%sMirror %s|url: %s|dist: %s|comp: %s|arch: %s|pkgs: %d%s", labelStart, repo.Name, repo.ArchiveRoot,
				repo.Distribution, strings.Join(repo.Components, ", "),
				strings.Join(repo.Architectures, ", "), repo.NumPackages(), labelEnd),
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
			"label": fmt.Sprintf("%sRepo %s|comment: %s|pkgs: %d%s", labelStart,
				repo.Name, repo.Comment, repo.NumPackages(), labelEnd),
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
			"label":     fmt.Sprintf("%sSnapshot %s|%s|pkgs: %d%s", labelStart,
				snapshot.Name, description, snapshot.NumPackages(), labelEnd),
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
			"label": fmt.Sprintf("%sPublished %s/%s|comp: %s|arch: %s%s", labelStart,
				repo.Prefix, repo.Distribution, strings.Join(repo.Components(), " "),
				strings.Join(repo.Architectures, ", "), labelEnd),
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
