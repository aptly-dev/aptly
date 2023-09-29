package deb

import (
	"fmt"
	"strings"

	"github.com/awalterschulze/gographviz"
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
		e := collectionFactory.RemoteRepoCollection().LoadComplete(repo)
		if e != nil {
			return e
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
		e := collectionFactory.LocalRepoCollection().LoadComplete(repo)
		if e != nil {
			return e
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
		e := collectionFactory.SnapshotCollection().LoadComplete(snapshot)
		if e != nil {
			return e
		}

		description := snapshot.Description
		if snapshot.SourceKind == SourceRemoteRepo {
			description = "Snapshot from repo"
		}

		graph.AddNode("aptly", snapshot.UUID, map[string]string{
			"shape":     "Mrecord",
			"style":     "filled",
			"fillcolor": "cadetblue1",
			"label": fmt.Sprintf("%sSnapshot %s|%s|pkgs: %d%s", labelStart,
				snapshot.Name, description, snapshot.NumPackages(), labelEnd),
		})

		if snapshot.SourceKind == SourceRemoteRepo || snapshot.SourceKind == SourceLocalRepo || snapshot.SourceKind == SourceSnapshot {
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
			"label": fmt.Sprintf("%sPublished %s|comp: %s|arch: %s%s", labelStart,
				repo.GetPath(), strings.Join(repo.Components(), " "),
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
