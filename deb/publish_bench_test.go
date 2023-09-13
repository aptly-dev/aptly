package deb

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/aptly-dev/aptly/database/goleveldb"
)

func BenchmarkListReferencedFiles(b *testing.B) {
	const defaultComponent = "main"
	const repoCount = 16
	const repoPackagesCount = 1024
	const uniqPackagesCount = 64

	tmpDir, err := os.MkdirTemp("", "aptly-bench")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := goleveldb.NewOpenDB(tmpDir)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	factory := NewCollectionFactory(db)
	packageCollection := factory.PackageCollection()
	repoCollection := factory.LocalRepoCollection()
	publishCollection := factory.PublishedRepoCollection()

	sharedRefs := NewPackageRefList()
	{
		transaction, err := db.OpenTransaction()
		if err != nil {
			b.Fatal(err)
		}

		for pkgIndex := 0; pkgIndex < repoPackagesCount-uniqPackagesCount; pkgIndex++ {
			p := &Package{
				Name:         fmt.Sprintf("pkg-shared_%d", pkgIndex),
				Version:      "1",
				Architecture: "amd64",
			}
			p.UpdateFiles(PackageFiles{PackageFile{
				Filename: fmt.Sprintf("pkg-shared_%d.deb", pkgIndex),
			}})

			packageCollection.UpdateInTransaction(p, transaction)
			sharedRefs.Refs = append(sharedRefs.Refs, p.Key(""))
		}

		sort.Sort(sharedRefs)

		if err := transaction.Commit(); err != nil {
			b.Fatal(err)
		}
	}

	for repoIndex := 0; repoIndex < repoCount; repoIndex++ {
		refs := NewPackageRefList()

		transaction, err := db.OpenTransaction()
		if err != nil {
			b.Fatal(err)
		}

		for pkgIndex := 0; pkgIndex < uniqPackagesCount; pkgIndex++ {
			p := &Package{
				Name:         fmt.Sprintf("pkg%d_%d", repoIndex, pkgIndex),
				Version:      "1",
				Architecture: "amd64",
			}
			p.UpdateFiles(PackageFiles{PackageFile{
				Filename: fmt.Sprintf("pkg%d_%d.deb", repoIndex, pkgIndex),
			}})

			packageCollection.UpdateInTransaction(p, transaction)
			refs.Refs = append(refs.Refs, p.Key(""))
		}

		if err := transaction.Commit(); err != nil {
			b.Fatal(err)
		}

		sort.Sort(refs)

		repo := NewLocalRepo(fmt.Sprintf("repo%d", repoIndex), "comment")
		repo.DefaultDistribution = fmt.Sprintf("dist%d", repoIndex)
		repo.DefaultComponent = defaultComponent
		repo.UpdateRefList(refs.Merge(sharedRefs, false, true))
		repoCollection.Add(repo)

		publish, err := NewPublishedRepo("", "test", "", nil, []string{defaultComponent}, []interface{}{repo}, factory)
		if err != nil {
			b.Fatal(err)
		}
		publishCollection.Add(publish)
	}

	db.CompactDB()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := publishCollection.listReferencedFilesByComponent("test", []string{defaultComponent}, factory, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
