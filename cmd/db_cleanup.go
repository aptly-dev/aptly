package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
	"sort"
)

// aptly db cleanup
func aptlyDbCleanup(cmd *commander.Command, args []string) error {
	var err error

	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	// collect information about references packages...
	existingPackageRefs := debian.NewPackageRefList()

	context.progress.Printf("Loading mirrors, local repos and snapshots...\n")
	repoCollection := debian.NewRemoteRepoCollection(context.database)
	err = repoCollection.ForEach(func(repo *debian.RemoteRepo) error {
		err := repoCollection.LoadComplete(repo)
		if err != nil {
			return err
		}
		if repo.RefList() != nil {
			existingPackageRefs = existingPackageRefs.Merge(repo.RefList(), false)
		}
		return nil
	})
	if err != nil {
		return err
	}

	localRepoCollection := debian.NewLocalRepoCollection(context.database)
	err = localRepoCollection.ForEach(func(repo *debian.LocalRepo) error {
		err := localRepoCollection.LoadComplete(repo)
		if err != nil {
			return err
		}
		if repo.RefList() != nil {
			existingPackageRefs = existingPackageRefs.Merge(repo.RefList(), false)
		}
		return nil
	})
	if err != nil {
		return err
	}

	snapshotCollection := debian.NewSnapshotCollection(context.database)
	err = snapshotCollection.ForEach(func(snapshot *debian.Snapshot) error {
		err := snapshotCollection.LoadComplete(snapshot)
		if err != nil {
			return err
		}
		existingPackageRefs = existingPackageRefs.Merge(snapshot.RefList(), false)
		return nil
	})
	if err != nil {
		return err
	}

	// ... and compare it to the list of all packages
	context.progress.Printf("Loading list of all packages...\n")
	packageCollection := debian.NewPackageCollection(context.database)
	allPackageRefs := packageCollection.AllPackageRefs()

	toDelete := allPackageRefs.Substract(existingPackageRefs)

	// delete packages that are no longer referenced
	context.progress.Printf("Deleting unreferenced packages (%d)...\n", toDelete.Len())

	context.database.StartBatch()
	err = toDelete.ForEach(func(ref []byte) error {
		return packageCollection.DeleteByKey(ref)
	})
	if err != nil {
		return err
	}

	err = context.database.FinishBatch()
	if err != nil {
		return fmt.Errorf("unable to write to DB: %s", err)
	}

	// now, build a list of files that should be present in Repository (package pool)
	context.progress.Printf("Building list of files referenced by packages...\n")
	referencedFiles := make([]string, 0, existingPackageRefs.Len())
	context.progress.InitBar(int64(existingPackageRefs.Len()), false)

	err = existingPackageRefs.ForEach(func(key []byte) error {
		pkg, err := packageCollection.ByKey(key)
		if err != nil {
			return err
		}
		paths, err := pkg.FilepathList(context.packagePool)
		if err != nil {
			return err
		}
		referencedFiles = append(referencedFiles, paths...)
		context.progress.AddBar(1)

		return nil
	})
	if err != nil {
		return err
	}

	sort.Strings(referencedFiles)
	context.progress.ShutdownBar()

	// build a list of files in the package pool
	context.progress.Printf("Building list of files in package pool...\n")
	existingFiles, err := context.packagePool.FilepathList(context.progress)
	if err != nil {
		return fmt.Errorf("unable to collect file paths: %s", err)
	}

	// find files which are in the pool but not referenced by packages
	filesToDelete := utils.StrSlicesSubstract(existingFiles, referencedFiles)

	// delete files that are no longer referenced
	context.progress.Printf("Deleting unreferenced files (%d)...\n", len(filesToDelete))

	if len(filesToDelete) > 0 {
		context.progress.InitBar(int64(len(filesToDelete)), false)
		totalSize := int64(0)
		for _, file := range filesToDelete {
			size, err := context.packagePool.Remove(file)
			if err != nil {
				return err
			}

			context.progress.AddBar(1)
			totalSize += size
		}
		context.progress.ShutdownBar()

		context.progress.Printf("Disk space freed: %.2f GiB...\n", float64(totalSize)/1024.0/1024.0/1024.0)
	}
	return err
}
