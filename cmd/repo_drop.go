package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
)

func aptlyRepoDrop(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return err
	}

	name := args[0]

	localRepoCollection := debian.NewLocalRepoCollection(context.database)
	repo, err := localRepoCollection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to drop: %s", err)
	}

	force := cmd.Flag.Lookup("force").Value.Get().(bool)
	if !force {
		snapshotCollection := debian.NewSnapshotCollection(context.database)
		snapshots := snapshotCollection.ByLocalRepoSource(repo)

		if len(snapshots) > 0 {
			fmt.Printf("Local repo `%s` was used to create following snapshots:\n", repo.Name)
			for _, snapshot := range snapshots {
				fmt.Printf(" * %s\n", snapshot)
			}

			return fmt.Errorf("won't delete local repo with snapshots, use -force to override")
		}
	}

	err = localRepoCollection.Drop(repo)
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
		Short:     "delete local repo",
		Long: `
Drop deletes information about local repo. Package data is not deleted
(it could be still used by other mirrors or snapshots).

ex:
  $ aptly repo drop local-repo
`,
		Flag: *flag.NewFlagSet("aptly-repo-drop", flag.ExitOnError),
	}

	cmd.Flag.Bool("force", false, "force local repo deletion even if used by snapshots")

	return cmd
}
