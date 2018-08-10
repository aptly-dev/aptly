package cmd

import (
	"fmt"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyRepoCreate(cmd *commander.Command, args []string) error {
	var err error
	if !(len(args) == 1 || (len(args) == 4 && args[1] == "from" && args[2] == "snapshot")) { // nolint: goconst
		cmd.Usage()
		return commander.ErrCommandError
	}

	repo := deb.NewLocalRepo(args[0], context.Flags().Lookup("comment").Value.String())
	repo.DefaultDistribution = context.Flags().Lookup("distribution").Value.String()
	repo.DefaultComponent = context.Flags().Lookup("component").Value.String()

	uploadersFile := context.Flags().Lookup("uploaders-file").Value.Get().(string)
	if uploadersFile != "" {
		repo.Uploaders, err = deb.NewUploadersFromFile(uploadersFile)
		if err != nil {
			return err
		}
	}

	collectionFactory := context.NewCollectionFactory()
	if len(args) == 4 {
		var snapshot *deb.Snapshot

		snapshot, err = collectionFactory.SnapshotCollection().ByName(args[3])
		if err != nil {
			return fmt.Errorf("unable to load source snapshot: %s", err)
		}

		err = collectionFactory.SnapshotCollection().LoadComplete(snapshot)
		if err != nil {
			return fmt.Errorf("unable to load source snapshot: %s", err)
		}

		repo.UpdateRefList(snapshot.RefList())
	}

	err = collectionFactory.LocalRepoCollection().Add(repo)
	if err != nil {
		return fmt.Errorf("unable to add local repo: %s", err)
	}

	fmt.Printf("\nLocal repo %s successfully added.\nYou can run 'aptly repo add %s ...' to add packages to repository.\n", repo, repo.Name)
	return err
}

func makeCmdRepoCreate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoCreate,
		UsageLine: "create <name> [ from snapshot <snapshot> ]",
		Short:     "create local repository",
		Long: `
Create local package repository. Repository would be empty when
created, packages could be added from files, copied or moved from
another local repository or imported from the mirror.

If local package repository is created from snapshot, repo initial
contents are copied from snapsot contents.

Example:

  $ aptly repo create testing

  $ aptly repo create mysql35 from snapshot mysql-35-2017
`,
		Flag: *flag.NewFlagSet("aptly-repo-create", flag.ExitOnError),
	}

	cmd.Flag.String("comment", "", "any text that would be used to described local repository")
	cmd.Flag.String("distribution", "", "default distribution when publishing")
	cmd.Flag.String("component", "main", "default component when publishing")
	cmd.Flag.String("uploaders-file", "", "uploaders.json to be used when including .changes into this repository")

	return cmd
}
