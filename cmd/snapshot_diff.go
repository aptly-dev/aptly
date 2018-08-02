package cmd

import (
	"fmt"

	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlySnapshotDiff(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 2 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	onlyMatching := context.Flags().Lookup("only-matching").Value.Get().(bool)
	collectionFactory := context.NewCollectionFactory()

	// Load <name-a> snapshot
	snapshotA, err := collectionFactory.SnapshotCollection().ByName(args[0])
	if err != nil {
		return fmt.Errorf("unable to load snapshot A: %s", err)
	}

	err = collectionFactory.SnapshotCollection().LoadComplete(snapshotA)
	if err != nil {
		return fmt.Errorf("unable to load snapshot A: %s", err)
	}

	// Load <name-b> snapshot
	snapshotB, err := collectionFactory.SnapshotCollection().ByName(args[1])
	if err != nil {
		return fmt.Errorf("unable to load snapshot B: %s", err)
	}

	err = collectionFactory.SnapshotCollection().LoadComplete(snapshotB)
	if err != nil {
		return fmt.Errorf("unable to load snapshot B: %s", err)
	}

	// Calculate diff
	diff, err := snapshotA.RefList().Diff(snapshotB.RefList(), collectionFactory.PackageCollection())
	if err != nil {
		return fmt.Errorf("unable to calculate diff: %s", err)
	}

	if len(diff) == 0 {
		context.Progress().Printf("Snapshots are identical.\n")
	} else {
		context.Progress().Printf("  Arch   | Package                                  | Version in A                             | Version in B\n")
		for _, pdiff := range diff {
			if onlyMatching && (pdiff.Left == nil || pdiff.Right == nil) {
				continue
			}

			var verA, verB, pkg, arch, code string

			if pdiff.Left == nil {
				verA = "-"
				verB = pdiff.Right.Version
				pkg = pdiff.Right.Name
				arch = pdiff.Right.Architecture
			} else {
				pkg = pdiff.Left.Name
				arch = pdiff.Left.Architecture
				verA = pdiff.Left.Version
				if pdiff.Right == nil {
					verB = "-"
				} else {
					verB = pdiff.Right.Version
				}
			}

			if pdiff.Left == nil {
				code = "@g+@|"
			} else {
				if pdiff.Right == nil {
					code = "@r-@|"
				} else {
					code = "@y!@|"
				}
			}

			context.Progress().ColoredPrintf(code+" %-6s | %-40s | %-40s | %-40s", arch, pkg, verA, verB)
		}
	}

	return err
}

func makeCmdSnapshotDiff() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotDiff,
		UsageLine: "diff <name-a> <name-b>",
		Short:     "difference between two snapshots",
		Long: `
Displays difference in packages between two snapshots. Snapshot is a list
of packages, so difference between snapshots is a difference between package
lists. Package could be either completely missing in one snapshot, or package
is present in both snapshots with different versions.

Example:

    $ aptly snapshot diff -only-matching wheezy-main wheezy-backports
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-diff", flag.ExitOnError),
	}

	cmd.Flag.Bool("only-matching", false, "display diff only for matching packages (don't display missing packages)")

	return cmd
}
