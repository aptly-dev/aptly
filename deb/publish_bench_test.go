package deb

import (
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/database/etcddb"
	"github.com/aptly-dev/aptly/database/goleveldb"
)

func BenchmarkListReferencedFiles(b *testing.B) {
	const defaultComponent = "main"
	const repoCount = 16
	const repoPackagesCount = 4 * 1024
	const uniqPackagesCount = repoPackagesCount / 16

	var db database.Storage
	if url := os.Getenv("APTLY_BENCH_ETCD"); len(url) > 0 {
		fmt.Println("using etcd at", url)

		var err error
		db, err = etcddb.NewDB(url)
		if err != nil {
			b.Fatal(err)
		}

		err = db.ProcessByPrefix([]byte{}, func(key []byte, value []byte) error {
			return fmt.Errorf("database is not empty, found key %s", string(key))
		})
		if err != nil {
			b.Fatal(err)
		}

		defer func() {
			batch := db.CreateBatch()
			_ = db.ProcessByPrefix([]byte{}, func(key []byte, value []byte) error {
				_ = batch.Delete(key)
				return nil
			})

			_ = batch.Write()
			_ = db.Close()
		}()
	} else {
		fmt.Println("using leveldb")

		tmpDir, err := os.MkdirTemp("", "aptly-bench")
		if err != nil {
			b.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(tmpDir) }()

		db, err = goleveldb.NewOpenDB(tmpDir)
		if err != nil {
			b.Fatal(err)
		}

		defer func() { _ = db.Close() }()
	}

	factory := NewCollectionFactory(db)
	packageCollection := factory.PackageCollection()
	repoCollection := factory.LocalRepoCollection()
	publishCollection := factory.PublishedRepoCollection()
	reflistCollection := factory.RefListCollection()

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

			err = packageCollection.UpdateInTransaction(p, transaction)
			if err != nil {
				b.Fatal(err)
			}
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

			err = packageCollection.UpdateInTransaction(p, transaction)
			if err != nil {
				b.Fatal(err)
			}
			refs.Refs = append(refs.Refs, p.Key(""))
		}

		if err := transaction.Commit(); err != nil {
			b.Fatal(err)
		}

		sort.Sort(refs)

		repo := NewLocalRepo(fmt.Sprintf("repo%d", repoIndex), "comment")
		repo.DefaultDistribution = fmt.Sprintf("dist%d", repoIndex)
		repo.DefaultComponent = defaultComponent
		merge := NewPackageRefSet(RefSetOptions{})
		merge.AddList(refs)
		merge.AddList(sharedRefs)
		repo.UpdateRefList(NewSplitRefListFromRefList(merge.ToRefList()))
		err = repoCollection.Add(repo, reflistCollection)
		if err != nil {
			b.Fatal(err)
		}

		publish, err := NewPublishedRepo("", "test", "", nil, []string{defaultComponent}, []interface{}{repo}, factory, false)
		if err != nil {
			b.Fatal(err)
		}
		err = publishCollection.Add(publish, reflistCollection)
		if err != nil {
			b.Fatal(err)
		}
	}

	_ = db.CompactDB()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := publishCollection.listReferencedFilesByComponent("test", []string{defaultComponent}, factory, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}
