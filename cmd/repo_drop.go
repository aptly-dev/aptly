package cmd

import (
	"fmt"

	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyRepoDrop(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	name := args[0]
	collectionFactory := context.NewCollectionFactory()

	repo, err := collectionFactory.LocalRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to drop: %s", err)
	}

	published := collectionFactory.PublishedRepoCollection().ByLocalRepo(repo)
	if len(published) > 0 {
		fmt.Printf("Local repo `%s` is published currently:\n", repo.Name)
		for _, repo := range published {
			err = collectionFactory.PublishedRepoCollection().LoadComplete(repo, collectionFactory)
			if err != nil {
				return fmt.Errorf("unable to load published: %s", err)
			}
			fmt.Printf(" * %s\n", repo)
		}

		return fmt.Errorf("unable to drop: local repo is published")
	}

	force := context.Flags().Lookup("force").Value.Get().(bool)
	if !force {
		snapshots := collectionFactory.SnapshotCollection().ByLocalRepoSource(repo)

		if len(snapshots) > 0 {
			fmt.Printf("Local repo `%s` was used to create following snapshots:\n", repo.Name)
			for _, snapshot := range snapshots {
				fmt.Printf(" * %s\n", snapshot)
			}

			return fmt.Errorf("won't delete local repo with snapshots, use -force to override")
		}
	}

	err = collectionFactory.LocalRepoCollection().Drop(repo)
	if err != nil {
		return fmt.Errorf("unable to drop: %s", err)
	}

	fmt.Printf("Local repo `%s` has been removed.\n", repo.Name)

	return err
}

func makeCmdRepoDrop() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoDrop,
		UsageLine: "drop <name>",
		Short:     "delete local repository",
		Long: `
Drop information about deletions from local repo. Package data is not deleted
(since it could be still used by other mirrors or snapshots).

Example:

  $ aptly repo drop local-repo
`,
		Flag: *flag.NewFlagSet("aptly-repo-drop", flag.ExitOnError),
	}

	cmd.Flag.Bool("force", false, "force local repo deletion even if used by snapshots")

	return cmd
}
