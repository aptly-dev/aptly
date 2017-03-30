package deb

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/database"
	"github.com/smira/go-uuid/uuid"
)

// ContentsIndex calculates mapping from files to packages, with sorting and aggregation
type ContentsIndex struct {
	db     database.Storage
	prefix []byte
}

// NewContentsIndex creates empty ContentsIndex
func NewContentsIndex(db database.Storage) *ContentsIndex {
	return &ContentsIndex{
		db:     db,
		prefix: []byte(uuid.New()),
	}
}

// Push adds package to contents index, calculating package contents as required
func (index *ContentsIndex) Push(p *Package, packagePool aptly.PackagePool, progress aptly.Progress) error {
	contents := p.Contents(packagePool, progress)
	qualifiedName := []byte(p.QualifiedName())

	for _, path := range contents {
		// for performance reasons we only write to leveldb during push.
		// merging of qualified names per path will be done in WriteTo
		err := index.db.Put(append(append(append(index.prefix, []byte(path)...), byte(0)), qualifiedName...), nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// Empty checks whether index contains no packages
func (index *ContentsIndex) Empty() bool {
	return !index.db.HasPrefix(index.prefix)
}

// WriteTo dumps sorted mapping of files to qualified package names
func (index *ContentsIndex) WriteTo(w io.Writer) (int64, error) {
	// For performance reasons push method wrote on key per path and package
	// in this method we now need to merge all packages which have the same path
	// and write it to contents index file

	var n int64

	nn, err := fmt.Fprintf(w, "%s %s\n", "FILE", "LOCATION")
	n += int64(nn)
	if err != nil {
		return n, err
	}

	prefixLen := len(index.prefix)

	var (
		currentPath []byte
		currentPkgs [][]byte
	)

	err = index.db.ProcessByPrefix(index.prefix, func(key []byte, value []byte) error {
		// cut prefix
		key = key[prefixLen:]

		i := bytes.Index(key, []byte{0})
		if i == -1 {
			return errors.New("corrupted index entry")
		}

		path := key[:i]
		pkg := key[i+1:]

		if !bytes.Equal(path, currentPath) {
			if currentPath != nil {
				nn, err = w.Write(append(currentPath, ' '))
				n += int64(nn)
				if err != nil {
					return err
				}

				nn, err = w.Write(bytes.Join(currentPkgs, []byte{','}))
				n += int64(nn)
				if err != nil {
					return err
				}

				nn, err = w.Write([]byte{'\n'})
				n += int64(nn)
				if err != nil {
					return err
				}
			}

			currentPath = append([]byte(nil), path...)
			currentPkgs = nil
		}

		currentPkgs = append(currentPkgs, append([]byte(nil), pkg...))

		return nil
	})

	if err != nil {
		return n, err
	}

	if currentPath != nil {
		nn, err = w.Write(append(currentPath, ' '))
		n += int64(nn)
		if err != nil {
			return n, err
		}

		nn, err = w.Write(bytes.Join(currentPkgs, []byte{','}))
		n += int64(nn)
		if err != nil {
			return n, err
		}

		nn, err = w.Write([]byte{'\n'})
		n += int64(nn)
	}

	return n, err
}
