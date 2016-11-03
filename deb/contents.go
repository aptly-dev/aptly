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

// Key generates unique identifier for contents index file with given path
func (index *ContentsIndex) Key(path string) []byte {
	refKey := index.repo.RefKey(index.component)
	return []byte(fmt.Sprintf("C%s_%s_%v$%s", refKey, index.architecture, index.udeb, path))
}

// Push adds package to contents index, calculating package contents as required
func (index *ContentsIndex) Push(p *Package, packagePool aptly.PackagePool) error {
	contents := p.Contents(packagePool)

	for _, path := range contents {
		var value []byte
		key := index.Key(path)
		dbValue, err := index.db.Get(key)

		if err != nil && err != database.ErrNotFound {
			return err
		}

		if dbValue == nil {
			value = []byte(p.QualifiedName())
		} else {
			name := "," + p.QualifiedName()
			value = []byte(string(dbValue) + name)
		}
		err = index.db.Put(key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

// Empty checks whether index contains no packages
func (index *ContentsIndex) Empty() bool {
	key := index.Key("")
	return !index.db.HasPrefix(key)
}

// WriteTo dumps sorted mapping of files to qualified package names
func (index *ContentsIndex) WriteTo(w io.Writer) (int64, error) {
	var n int64

	nn, err := fmt.Fprintf(w, "%s %s\n", "FILE", "LOCATION")
	n += int64(nn)
	if err != nil {
		return n, err
	}

	prefix := index.Key("")
	err = index.db.ProcessByPrefix(prefix, func(key []byte, value []byte) error {
		parts := strings.Split(string(key), "$")
		path := parts[len(parts)-1]
		nn, err = fmt.Fprintf(w, "%s %s\n", path, string(value))
		n += int64(nn)
		return err
	})

	return n, err
}
