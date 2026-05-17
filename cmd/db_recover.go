package cmd

import (
	"fmt"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"

	"github.com/aptly-dev/aptly/database/goleveldb"
)

// aptly db recover
func aptlyDBRecover(cmd *commander.Command, args []string) error {
	var err error

	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	context.Progress().Printf("Recovering database...\n")
	if err = goleveldb.RecoverDB(context.DBPath()); err != nil {
		return err
	}

	context.Progress().Printf("Checking database integrity...\n")
	err = checkIntegrity()

	return err
}

func makeCmdDBRecover() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyDBRecover,
		UsageLine: "recover",
		Short:     "recover DB after crash",
		Long: `
Database recover does its' best to recover the database after a crash.
It is recommended to backup the DB before running recover.

Example:

  $ aptly db recover
`,
	}

	return cmd
}

func checkIntegrity() error {
	return context.NewCollectionFactory().LocalRepoCollection().ForEach(checkRepo)
}

func checkRepo(repo *deb.LocalRepo) error {
	collectionFactory := context.NewCollectionFactory()
	repos := collectionFactory.LocalRepoCollection()

	err := repos.LoadComplete(repo)
	if err != nil {
		// If we can't load the repo, it might be severely corrupted
		// Log the error but continue with other repos
		context.Progress().Printf("Warning: Cannot load repo %q: %s\n", repo.Name, err)
		return nil
	}

	// Check if RefList is nil (severe corruption case)
	refList := repo.RefList()
	if refList == nil {
		context.Progress().Printf("Warning: Repo %q has no reference list (severely corrupted), initializing empty list\n", repo.Name)
		// Initialize with empty reflist
		repo.UpdateRefList(deb.NewPackageRefList())
		if err = repos.Update(repo); err != nil {
			return fmt.Errorf("update repo with empty reflist: %w", err)
		}
		return nil
	}

	dangling, err := deb.FindDanglingReferences(refList, collectionFactory.PackageCollection())
	if err != nil {
		// If we can't find dangling references, log but continue
		context.Progress().Printf("Warning: Cannot check dangling references for repo %q: %s\n", repo.Name, err)
		return nil
	}

	if dangling != nil && len(dangling.Refs) > 0 {
		for _, ref := range dangling.Refs {
			context.Progress().Printf("Removing dangling database reference %q\n", ref)
		}

		repo.UpdateRefList(refList.Subtract(dangling))

		if err = repos.Update(repo); err != nil {
			return fmt.Errorf("update repo: %w", err)
		}
	}

	return nil
}
