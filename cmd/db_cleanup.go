package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/utils"
	"github.com/smira/commander"
	"sort"
)

// aptly db cleanup
func aptlyDbCleanup(cmd *commander.Command, args []string) error {
	var err error

	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	// collect information about references packages...
	existingPackageRefs := deb.NewPackageRefList()

	context.Progress().Printf("Loading mirrors, local repos and snapshots...\n")
	err = context.CollectionFactory().RemoteRepoCollection().ForEach(func(repo *deb.RemoteRepo) error {
		err := context.CollectionFactory().RemoteRepoCollection().LoadComplete(repo)
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

	err = context.CollectionFactory().LocalRepoCollection().ForEach(func(repo *deb.LocalRepo) error {
		err := context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
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

	err = context.CollectionFactory().SnapshotCollection().ForEach(func(snapshot *deb.Snapshot) error {
		err := context.CollectionFactory().SnapshotCollection().LoadComplete(snapshot)
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
	context.Progress().Printf("Loading list of all packages...\n")
	allPackageRefs := context.CollectionFactory().PackageCollection().AllPackageRefs()

	toDelete := allPackageRefs.Substract(existingPackageRefs)

	// delete packages that are no longer referenced
	context.Progress().Printf("Deleting unreferenced packages (%d)...\n", toDelete.Len())

	// database can't err as collection factory already constructed
	db, _ := context.Database()
	db.StartBatch()
	err = toDelete.ForEach(func(ref []byte) error {
		return context.CollectionFactory().PackageCollection().DeleteByKey(ref)
	})
	if err != nil {
		return err
	}

	err = db.FinishBatch()
	if err != nil {
		return fmt.Errorf("unable to write to DB: %s", err)
	}

	// now, build a list of files that should be present in Repository (package pool)
	context.Progress().Printf("Building list of files referenced by packages...\n")
	referencedFiles := make([]string, 0, existingPackageRefs.Len())
	context.Progress().InitBar(int64(existingPackageRefs.Len()), false)

	err = existingPackageRefs.ForEach(func(key []byte) error {
		pkg, err2 := context.CollectionFactory().PackageCollection().ByKey(key)
		if err2 != nil {
			return err2
		}
		paths, err2 := pkg.FilepathList(context.PackagePool())
		if err2 != nil {
			return err2
		}
		referencedFiles = append(referencedFiles, paths...)
		context.Progress().AddBar(1)

		return nil
	})
	if err != nil {
		return err
	}

	sort.Strings(referencedFiles)
	context.Progress().ShutdownBar()

	// build a list of files in the package pool
	context.Progress().Printf("Building list of files in package pool...\n")
	existingFiles, err := context.PackagePool().FilepathList(context.Progress())
	if err != nil {
		return fmt.Errorf("unable to collect file paths: %s", err)
	}

	// find files which are in the pool but not referenced by packages
	filesToDelete := utils.StrSlicesSubstract(existingFiles, referencedFiles)

	// delete files that are no longer referenced
	context.Progress().Printf("Deleting unreferenced files (%d)...\n", len(filesToDelete))

	if len(filesToDelete) > 0 {
		context.Progress().InitBar(int64(len(filesToDelete)), false)

		var size, totalSize int64
		for _, file := range filesToDelete {
			size, err = context.PackagePool().Remove(file)
			if err != nil {
				return err
			}

			context.Progress().AddBar(1)
			totalSize += size
		}
		context.Progress().ShutdownBar()

		context.Progress().Printf("Disk space freed: %s...\n", utils.HumanBytes(totalSize))
	}

	context.Progress().Printf("Compacting database...\n")
	err = db.CompactDB()

	return err
}

func makeCmdDbCleanup() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyDbCleanup,
		UsageLine: "cleanup",
		Short:     "cleanup DB and package pool",
		Long: `
Database cleanup removes information about unreferenced packages and removes
files in the package pool that aren't used by packages anymore

Example:

  $ aptly db cleanup
`,
	}

	return cmd
}
