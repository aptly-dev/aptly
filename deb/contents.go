package deb

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/database"
	"io"
	"strings"
)

// ContentsIndex calculates mapping from files to packages, with sorting and aggregation
type ContentsIndex struct {
	db           database.Storage
	repo         PublishedRepo
	component    string
	architecture string
	udeb         bool
}

// NewContentsIndex creates empty ContentsIndex
func NewContentsIndex(db database.Storage, repo PublishedRepo, component string, architecture string, udeb bool) *ContentsIndex {
	return &ContentsIndex{db: db, repo: repo, component: component, architecture: architecture}
}

// Key generates unique identifier for contents index file with given path and package name
func (index *ContentsIndex) Key(path string, pkg string) []byte {
	refKey := index.repo.RefKey(index.component)
	// For prefix to still work when pkg is empty only append
	// separator when pkg is set
	if pkg != "" {
		pkg = "$" + pkg
	}
	return []byte(fmt.Sprintf("xI%s_%s_%v$%s%s", refKey, index.architecture, index.udeb, path, pkg))
}

// Push adds package to contents index, calculating package contents as required
func (index *ContentsIndex) Push(p *Package, packagePool aptly.PackagePool) error {
	contents := p.Contents(packagePool)

	index.db.StartBatch()
	defer index.db.FinishBatch()

	for _, path := range contents {
		// for performance reasons we only write to leveldb during push.
		// Merging of qualified names per path will be done in WriteTo
		key := index.Key(path, p.QualifiedName())

		err := index.db.Put(key, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// Empty checks whether index contains no packages
func (index *ContentsIndex) Empty() bool {
	key := index.Key("", "")
	return !index.db.HasPrefix(key)
}

// WriteTo dumps sorted mapping of files to qualified package names
func (index *ContentsIndex) WriteTo(w io.Writer) (int64, error) {
	// For performance reasons push method wrote on key per path and package
	// in this method we now need to merge all pkg with have the same path
	// and write it to contents index file
	var n int64

	nn, err := fmt.Fprintf(w, "%s %s\n", "FILE", "LOCATION")
	n += int64(nn)
	if err != nil {
		return n, err
	}

	prefix := index.Key("", "")
	currentPath := ""
	currentPkgs := make([]string, 0, 1)

	err = index.db.ProcessByPrefix(prefix, func(key []byte, value []byte) error {
		parts := strings.Split(string(key), "$")
		path := parts[len(parts)-2]
		pkg := parts[len(parts)-1]

		if currentPath != "" && currentPath != path {
			nn, err = fmt.Fprintf(w, "%s %s\n", currentPath, strings.Join(currentPkgs, ","))
			n += int64(nn)
			if err != nil {
				return err
			}
			currentPkgs = make([]string, 0, 1)
		}

		currentPath = path
		currentPkgs = append(currentPkgs, pkg)
		return nil
	})

	if err != nil {
		return n, err
	}

	if len(currentPkgs) > 0 && currentPath != "" {
		nn, err = fmt.Fprintf(w, "%s %s\n", currentPath, strings.Join(currentPkgs, ","))
		n += int64(nn)
	}

	return n, err
}
