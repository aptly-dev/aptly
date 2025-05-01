package deb

import (
	"errors"
	"fmt"

	"github.com/aptly-dev/aptly/database"
)

// FindDanglingReferences finds references that exist in the given PackageRefList, but not in the given PackageCollection.
// It returns all such references, so they can be removed from the database.
func FindDanglingReferences(reflist *PackageRefList, packages *PackageCollection) (dangling *PackageRefList, err error) {
	dangling = &PackageRefList{}

	err = reflist.ForEach(func(key []byte) error {
		ok, err := isDangling(packages, key)
		if err != nil {
			return err
		}

		if ok {
			dangling.Refs = append(dangling.Refs, key)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return dangling, nil
}

func isDangling(packages *PackageCollection, key []byte) (bool, error) {
	_, err := packages.ByKey(key)
	if errors.Is(err, database.ErrNotFound) {
		return true, nil
	}

	if err != nil {
		return false, fmt.Errorf("get reference %q: %w", key, err)
	}

	return false, nil
}
