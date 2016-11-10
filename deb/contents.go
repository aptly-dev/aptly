package deb

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/utils"
	"io"
	"sort"
	"strings"
)

// ContentsIndex calculates mapping from files to packages, with sorting and aggregation
type ContentsIndex struct {
	index map[string][]*Package
}

// NewContentsIndex creates empty ContentsIndex
func NewContentsIndex() *ContentsIndex {
	return &ContentsIndex{
		index: make(map[string][]*Package),
	}
}

// Push adds package to contents index, calculating package contents as required
func (index *ContentsIndex) Push(p *Package, packagePool aptly.PackagePool) {
	contents := p.Contents(packagePool)

	for _, path := range contents {
		index.index[path] = append(index.index[path], p)
	}
}

// Empty checks whether index contains no packages
func (index *ContentsIndex) Empty() bool {
	return len(index.index) == 0
}

// WriteTo dumps sorted mapping of files to qualified package names
func (index *ContentsIndex) WriteTo(w io.Writer) (int64, error) {
	var n int64

	paths := make([]string, len(index.index))

	i := 0
	for path := range index.index {
		paths[i] = path
		i++
	}

	sort.Strings(paths)

	nn, err := fmt.Fprintf(w, "%s %s\n", "FILE", "LOCATION")
	n += int64(nn)
	if err != nil {
		return n, err
	}

	for _, path := range paths {
		packages := index.index[path]
		parts := make([]string, 0, len(packages))
		for i := range packages {
			name := packages[i].QualifiedName()
			if !utils.StrSliceHasItem(parts, name) {
				parts = append(parts, name)
			}
		}
		nn, err = fmt.Fprintf(w, "%s %s\n", path, strings.Join(parts, ","))
		n += int64(nn)
		if err != nil {
			return n, err
		}
	}

	return n, nil
}
