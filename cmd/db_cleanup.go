package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/utils"
	"github.com/smira/commander"
	"sort"
	"strings"
)

// aptly db cleanup
func aptlyDbCleanup(cmd *commander.Command, args []string) error {
	var err error

	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	verbose := context.Flags().Lookup("verbose").Value.Get().(bool)
	dryRun := context.Flags().Lookup("dry-run").Value.Get().(bool)

	// collect information about references packages...
	existingPackageRefs := deb.NewPackageRefList()

	// used only in verbose mode to report package use source
	packageRefSources := map[string][]string{}

	context.Progress().ColoredPrintf("@{w!}Loading mirrors, local repos, snapshots and published repos...@|")
	if verbose {
		context.Progress().ColoredPrintf("@{y}Loading mirrors:@|")
	}
	err = context.CollectionFactory().RemoteRepoCollection().ForEach(func(repo *deb.RemoteRepo) error {
		if verbose {
			context.Progress().ColoredPrintf("- @{g}%s@|", repo.Name)
		}

		err := context.CollectionFactory().RemoteRepoCollection().LoadComplete(repo)
		if err != nil {
			return err
		}
		if repo.RefList() != nil {
			existingPackageRefs = existingPackageRefs.Merge(repo.RefList(), false, true)

			if verbose {
				description := fmt.Sprintf("mirror %s", repo.Name)
				repo.RefList().ForEach(func(key []byte) error {
					packageRefSources[string(key)] = append(packageRefSources[string(key)], description)
					return nil
				})
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if verbose {
		context.Progress().ColoredPrintf("@{y}Loading local repos:@|")
	}
	err = context.CollectionFactory().LocalRepoCollection().ForEach(func(repo *deb.LocalRepo) error {
		if verbose {
			context.Progress().ColoredPrintf("- @{g}%s@|", repo.Name)
		}

		err := context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
		if err != nil {
			return err
		}

		if repo.RefList() != nil {
			existingPackageRefs = existingPackageRefs.Merge(repo.RefList(), false, true)

			if verbose {
				description := fmt.Sprintf("local repo %s", repo.Name)
				repo.RefList().ForEach(func(key []byte) error {
					packageRefSources[string(key)] = append(packageRefSources[string(key)], description)
					return nil
				})
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	if verbose {
		context.Progress().ColoredPrintf("@{y}Loading snapshots:@|")
	}
	err = context.CollectionFactory().SnapshotCollection().ForEach(func(snapshot *deb.Snapshot) error {
		if verbose {
			context.Progress().ColoredPrintf("- @{g}%s@|", snapshot.Name)
		}

		err := context.CollectionFactory().SnapshotCollection().LoadComplete(snapshot)
		if err != nil {
			return err
		}

		existingPackageRefs = existingPackageRefs.Merge(snapshot.RefList(), false, true)

		if verbose {
			description := fmt.Sprintf("snapshot %s", snapshot.Name)
			snapshot.RefList().ForEach(func(key []byte) error {
				packageRefSources[string(key)] = append(packageRefSources[string(key)], description)
				return nil
			})
		}
		return nil
	})
	if err != nil {
		return err
	}

	if verbose {
		context.Progress().ColoredPrintf("@{y}Loading published repositories:@|")
	}
	err = context.CollectionFactory().PublishedRepoCollection().ForEach(func(published *deb.PublishedRepo) error {
		if verbose {
			context.Progress().ColoredPrintf("- @{g}%s:%s/%s{|}", published.Storage, published.Prefix, published.Distribution)
		}
		if published.SourceKind != "local" {
			return nil
		}
		err := context.CollectionFactory().PublishedRepoCollection().LoadComplete(published, context.CollectionFactory())
		if err != nil {
			return err
		}

		for _, component := range published.Components() {
			existingPackageRefs = existingPackageRefs.Merge(published.RefList(component), false, true)
			if verbose {
				description := fmt.Sprintf("published repository %s:%s/%s component %s",
					published.Storage, published.Prefix, published.Distribution, component)
				published.RefList(component).ForEach(func(key []byte) error {
					packageRefSources[string(key)] = append(packageRefSources[string(key)], description)
					return nil
				})
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// ... and compare it to the list of all packages
	context.Progress().ColoredPrintf("@{w!}Loading list of all packages...@|")
	allPackageRefs := context.CollectionFactory().PackageCollection().AllPackageRefs()

	toDelete := allPackageRefs.Substract(existingPackageRefs)

	// delete packages that are no longer referenced
	context.Progress().ColoredPrintf("@{r!}Deleting unreferenced packages (%d)...@|", toDelete.Len())

	// database can't err as collection factory already constructed
	db, _ := context.Database()

	if toDelete.Len() > 0 {
		if verbose {
			context.Progress().ColoredPrintf("@{r}List of package keys to delete:@|")
			err = toDelete.ForEach(func(ref []byte) error {
				context.Progress().ColoredPrintf(" - @{r}%s@|", string(ref))
				return nil
			})
			if err != nil {
				return err
			}
		}

		if !dryRun {
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
		} else {
			context.Progress().ColoredPrintf("@{y!}Skipped deletion, as -dry-run has been requested.@|")
		}
	}

	// now, build a list of files that should be present in Repository (package pool)
	context.Progress().ColoredPrintf("@{w!}Building list of files referenced by packages...@|")
	referencedFiles := make([]string, 0, existingPackageRefs.Len())
	context.Progress().InitBar(int64(existingPackageRefs.Len()), false)

	err = existingPackageRefs.ForEach(func(key []byte) error {
		pkg, err2 := context.CollectionFactory().PackageCollection().ByKey(key)
		if err2 != nil {
			tail := ""
			if verbose {
				tail = fmt.Sprintf(" (sources: %s)", strings.Join(packageRefSources[string(key)], ", "))
			}
			if dryRun {
				context.Progress().ColoredPrintf("@{r!}Unresolvable package reference, skipping (-dry-run): %s: %s%s",
					string(key), err2, tail)
				return nil
			}
			return fmt.Errorf("unable to load package %s: %s%s", string(key), err2, tail)
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
	context.Progress().ColoredPrintf("@{w!}Building list of files in package pool...@|")
	existingFiles, err := context.PackagePool().FilepathList(context.Progress())
	if err != nil {
		return fmt.Errorf("unable to collect file paths: %s", err)
	}

	// find files which are in the pool but not referenced by packages
	filesToDelete := utils.StrSlicesSubstract(existingFiles, referencedFiles)

	// delete files that are no longer referenced
	context.Progress().ColoredPrintf("@{r!}Deleting unreferenced files (%d)...@|", len(filesToDelete))

	if len(filesToDelete) > 0 {
		if verbose {
			context.Progress().ColoredPrintf("@{r}List of files to be deleted:@|")
			for _, file := range filesToDelete {
				context.Progress().ColoredPrintf(" - @{r}%s@|", file)
			}
		}

		if !dryRun {
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

			context.Progress().ColoredPrintf("@{w!}Disk space freed: %s...@|", utils.HumanBytes(totalSize))
		} else {
			context.Progress().ColoredPrintf("@{y!}Skipped file deletion, as -dry-run has been requested.@|")
		}
	}

	if !dryRun {
		context.Progress().ColoredPrintf("@{w!}Compacting database...@|")
		err = db.CompactDB()
	} else {
		context.Progress().ColoredPrintf("@{y!}Skipped DB compaction, as -dry-run has been requested.@|")
	}

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

	cmd.Flag.Bool("verbose", false, "be verbose when loading objects/removing them")
	cmd.Flag.Bool("dry-run", false, "don't delete anything")

	return cmd
}
