package api

import (
	"fmt"
	"sort"

	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/task"
	"github.com/aptly-dev/aptly/utils"
	"github.com/gin-gonic/gin"
)

// POST /api/db/cleanup
func apiDbCleanup(c *gin.Context) {

	resources := []string{string(task.AllResourcesKey)}
	currTask, conflictErr := runTaskInBackground("Clean up db", resources, func(out *task.Output, detail *task.Detail) error {
		var err error

		collectionFactory := context.NewCollectionFactory()

		// collect information about referenced packages...
		existingPackageRefs := deb.NewPackageRefList()

		out.Printf("Loading mirrors, local repos, snapshots and published repos...")
		err = collectionFactory.RemoteRepoCollection().ForEach(func(repo *deb.RemoteRepo) error {
			e := collectionFactory.RemoteRepoCollection().LoadComplete(repo)
			if e != nil {
				return e
			}
			if repo.RefList() != nil {
				existingPackageRefs = existingPackageRefs.Merge(repo.RefList(), false, true)
			}

			return nil
		})
		if err != nil {
			return err
		}

		err = collectionFactory.LocalRepoCollection().ForEach(func(repo *deb.LocalRepo) error {
			e := collectionFactory.LocalRepoCollection().LoadComplete(repo)
			if e != nil {
				return e
			}

			if repo.RefList() != nil {
				existingPackageRefs = existingPackageRefs.Merge(repo.RefList(), false, true)
			}

			return nil
		})
		if err != nil {
			return err
		}

		err = collectionFactory.SnapshotCollection().ForEach(func(snapshot *deb.Snapshot) error {
			e := collectionFactory.SnapshotCollection().LoadComplete(snapshot)
			if e != nil {
				return e
			}

			existingPackageRefs = existingPackageRefs.Merge(snapshot.RefList(), false, true)

			return nil
		})
		if err != nil {
			return err
		}

		err = collectionFactory.PublishedRepoCollection().ForEach(func(published *deb.PublishedRepo) error {
			if published.SourceKind != deb.SourceLocalRepo {
				return nil
			}
			e := collectionFactory.PublishedRepoCollection().LoadComplete(published, collectionFactory)
			if e != nil {
				return e
			}

			for _, component := range published.Components() {
				existingPackageRefs = existingPackageRefs.Merge(published.RefList(component), false, true)
			}
			return nil
		})
		if err != nil {
			return err
		}

		// ... and compare it to the list of all packages
		out.Printf("Loading list of all packages...")
		allPackageRefs := collectionFactory.PackageCollection().AllPackageRefs()

		toDelete := allPackageRefs.Subtract(existingPackageRefs)

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
				return fmt.Errorf("unable to write to DB: %s", err)
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
			return err
		}

		sort.Strings(referencedFiles)

		// build a list of files in the package pool
		out.Printf("Building list of files in package pool...")
		existingFiles, err := context.PackagePool().FilepathList(out)
		if err != nil {
			return fmt.Errorf("unable to collect file paths: %s", err)
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
					return err
				}

				taskDetail.RemainingNumberOfPackagesToDelete--
				detail.Store(taskDetail)
				totalSize += size
			}

			out.Printf("Disk space freed: %s...", utils.HumanBytes(totalSize))
		}

		out.Printf("Compacting database...")
		return db.CompactDB()
	})

	if conflictErr != nil {
		c.AbortWithError(409, conflictErr)
		return
	}

	c.JSON(202, currTask)
}
