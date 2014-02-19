package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
)

func aptlyMirrorDrop(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return err
	}

	name := args[0]

	repoCollection := debian.NewRemoteRepoCollection(context.database)
	repo, err := repoCollection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to drop: %s", err)
	}

	force := cmd.Flag.Lookup("force").Value.Get().(bool)
	if !force {
		snapshotCollection := debian.NewSnapshotCollection(context.database)
		snapshots := snapshotCollection.ByRemoteRepoSource(repo)

		if len(snapshots) > 0 {
			fmt.Printf("Mirror `%s` was used to create following snapshots:\n", repo.Name)
			for _, snapshot := range snapshots {
				fmt.Printf(" * %s\n", snapshot)
			}

			return fmt.Errorf("won't delete mirror with snapshots, use -force to override")
		}
	}

	err = repoCollection.Drop(repo)
	if err != nil {
		return fmt.Errorf("unable to drop: %s", err)
	}

	fmt.Printf("Mirror `%s` has been removed.\n", repo.Name)

	return err
}

func makeCmdMirrorDrop() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorDrop,
		UsageLine: "drop <name>",
		Short:     "delete remote repository mirror",
		Long: `
Drop deletes information about remote repository mirror. Package data is not deleted
(it could be still used by other mirrors or snapshots).

ex:
  $ aptly mirror drop wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-mirror-drop", flag.ExitOnError),
	}

	cmd.Flag.Bool("force", false, "force mirror deletion even if used by snapshots")

	return cmd
}
