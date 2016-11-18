package cmd

import (
	"fmt"

	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyMirrorDrop(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	name := args[0]
	collectionFactory := context.NewCollectionFactory()

	repo, err := collectionFactory.RemoteRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to drop: %s", err)
	}

	err = repo.CheckLock()
	if err != nil {
		return fmt.Errorf("unable to drop: %s", err)
	}

	force := context.Flags().Lookup("force").Value.Get().(bool)
	if !force {
		snapshots := collectionFactory.SnapshotCollection().ByRemoteRepoSource(repo)

		if len(snapshots) > 0 {
			fmt.Printf("Mirror `%s` was used to create following snapshots:\n", repo.Name)
			for _, snapshot := range snapshots {
				fmt.Printf(" * %s\n", snapshot)
			}

			return fmt.Errorf("won't delete mirror with snapshots, use -force to override")
		}
	}

	err = collectionFactory.RemoteRepoCollection().Drop(repo)
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
		Short:     "delete mirror",
		Long: `
Drop deletes information about remote repository mirror <name>. Package data is not deleted
(since it could still be used by other mirrors or snapshots).  If mirror is used as source
to create a snapshot, aptly would refuse to delete such mirror, use flag -force to override.

Example:

  $ aptly mirror drop wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-mirror-drop", flag.ExitOnError),
	}

	cmd.Flag.Bool("force", false, "force mirror deletion even if used by snapshots")

	return cmd
}
