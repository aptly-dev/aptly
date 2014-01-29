package main

import (
	"fmt"
	"github.com/smira/aptly/debian"
)

//ListPackagesRefList shows list of packages in PackageRefList
func ListPackagesRefList(reflist *debian.PackageRefList) (err error) {
	fmt.Printf("Packages:\n")

	packageCollection := debian.NewPackageCollection(context.database)

	err = reflist.ForEach(func(key []byte) error {
		p, err := packageCollection.ByKey(key)
		if err != nil {
			return err
		}
		fmt.Printf("  %s\n", p)
		return nil
	})
	if err != nil {
		return fmt.Errorf("unable to load packages: %s", err)
	}

	return
}
