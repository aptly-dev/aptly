package api

import (
	"fmt"
	"sort"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/task"
	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
)

// POST /api/db/cleanup
func apiDbCleanup(c *gin.Context) {

	resources := []string{string(task.AllResourcesKey)}
	maybeRunTaskInBackground(c, "Clean up db", resources, func(out aptly.Progress, detail *task.Detail) (*task.ProcessReturnValue, error) {
		var err error

		collectionFactory := context.NewCollectionFactory()

		// collect information about referenced packages and their reflist buckets...
		existingPackageRefs := deb.NewSplitRefList()
		existingBuckets := deb.NewRefListDigestSet()

		reflistMigration := collectionFactory.RefListCollection().NewMigration()

		out.Printf("Loading mirrors, local repos, snapshots and published repos...")
		err = collectionFactory.RemoteRepoCollection().ForEach(func(repo *deb.RemoteRepo) error {
			sl := deb.NewSplitRefList()
			e := collectionFactory.RefListCollection().LoadCompleteAndMigrate(sl, repo.RefKey(), reflistMigration)
			if e != nil && e != database.ErrNotFound {
				return e
			}

			existingPackageRefs = existingPackageRefs.Merge(sl, false, true)
			existingBuckets.AddAllInRefList(sl)

			return nil
		})
		if err != nil {
			return nil, err
		}

		err = collectionFactory.LocalRepoCollection().ForEach(func(repo *deb.LocalRepo) error {
			sl := deb.NewSplitRefList()
			e := collectionFactory.RefListCollection().LoadCompleteAndMigrate(sl, repo.RefKey(), reflistMigration)
			if e != nil && e != database.ErrNotFound {
				return e
			}

			existingPackageRefs = existingPackageRefs.Merge(sl, false, true)
			existingBuckets.AddAllInRefList(sl)

			return nil
		})
		if err != nil {
			return nil, err
		}

		err = collectionFactory.SnapshotCollection().ForEach(func(snapshot *deb.Snapshot) error {
			sl := deb.NewSplitRefList()
			e := collectionFactory.RefListCollection().LoadCompleteAndMigrate(sl, snapshot.RefKey(), reflistMigration)
			if e != nil {
				return e
			}

			existingPackageRefs = existingPackageRefs.Merge(sl, false, true)
			existingBuckets.AddAllInRefList(sl)

			return nil
		})
		if err != nil {
			return nil, err
		}

		err = collectionFactory.PublishedRepoCollection().ForEach(func(published *deb.PublishedRepo) error {
			if published.SourceKind != deb.SourceLocalRepo {
				return nil
			}

			for _, component := range published.Components() {
				sl := deb.NewSplitRefList()
				e := collectionFactory.RefListCollection().LoadCompleteAndMigrate(sl, published.RefKey(component), reflistMigration)
				if e != nil {
					return e
				}

				existingPackageRefs = existingPackageRefs.Merge(sl, false, true)
				existingBuckets.AddAllInRefList(sl)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		err = reflistMigration.Flush()
		if err != nil {
			return nil, err
		}
		if stats := reflistMigration.Stats(); stats.Reflists > 0 {
			out.Printf("Split %d reflist(s) into %d bucket(s) (%d segment(s))",
				stats.Reflists, stats.Buckets, stats.Segments)
		}

		// ... and compare it to the list of all packages
		out.Printf("Loading list of all packages...")
		allPackageRefs := collectionFactory.PackageCollection().AllPackageRefs()

		toDelete := allPackageRefs.Subtract(existingPackageRefs.Flatten())

		// delete packages that are no longer referenced
		out.Printf("Deleting unreferenced packages (%d)...", toDelete.Len())

		// database can't err as collection factory already constructed
		db, _ := context.Database()

		if toDelete.Len() > 0 {
			batch := db.CreateBatch()
			toDelete.ForEach(func(ref []byte) error {
				collectionFactory.PackageCollection().DeleteByKey(ref, batch)
				return nil
			})

			err = batch.Write()
			if err != nil {
				return nil, fmt.Errorf("unable to write to DB: %s", err)
			}
		}

		bucketsToDelete, err := collectionFactory.RefListCollection().AllBucketDigests()
		if err != nil {
			return nil, err
		}

		bucketsToDelete.RemoveAll(existingBuckets)

		out.Printf("Deleting unreferenced reflist buckets (%d)...", bucketsToDelete.Len())
		if bucketsToDelete.Len() > 0 {
			batch := db.CreateBatch()
			err := bucketsToDelete.ForEach(func(digest []byte) error {
				return collectionFactory.RefListCollection().UnsafeDropBucket(digest, batch)
			})
			if err != nil {
				return nil, err
			}

			if err := batch.Write(); err != nil {
				return nil, err
			}
		}

		// now, build a list of files that should be present in Repository (package pool)
		out.Printf("Building list of files referenced by packages...")
		referencedFiles := make([]string, 0, existingPackageRefs.Len())

		err = existingPackageRefs.ForEach(func(key []byte) error {
			pkg, err2 := collectionFactory.PackageCollection().ByKey(key)
			if err2 != nil {
				tail := ""
				return fmt.Errorf("unable to load package %s: %s%s", string(key), err2, tail)
			}
			paths, err2 := pkg.FilepathList(context.PackagePool())
			if err2 != nil {
				return err2
			}
			referencedFiles = append(referencedFiles, paths...)

			return nil
		})
		if err != nil {
			return nil, err
		}

		sort.Strings(referencedFiles)

		// build a list of files in the package pool
		out.Printf("Building list of files in package pool...")
		existingFiles, err := context.PackagePool().FilepathList(out)
		if err != nil {
			return nil, fmt.Errorf("unable to collect file paths: %s", err)
		}

		// find files which are in the pool but not referenced by packages
		filesToDelete := utils.StrSlicesSubstract(existingFiles, referencedFiles)

		// delete files that are no longer referenced
		out.Printf("Deleting unreferenced files (%d)...", len(filesToDelete))

		countFilesToDelete := len(filesToDelete)
		taskDetail := struct {
			TotalNumberOfPackagesToDelete     int
			RemainingNumberOfPackagesToDelete int
		}{
			countFilesToDelete, countFilesToDelete,
		}
		detail.Store(taskDetail)

		if countFilesToDelete > 0 {
			var size, totalSize int64
			for _, file := range filesToDelete {
				size, err = context.PackagePool().Remove(file)
				if err != nil {
					return nil, err
				}

				taskDetail.RemainingNumberOfPackagesToDelete--
				detail.Store(taskDetail)
				totalSize += size
			}

			out.Printf("Disk space freed: %s...", utils.HumanBytes(totalSize))
		}

		out.Printf("Compacting database...")
		return nil, db.CompactDB()
	})
}
