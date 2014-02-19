package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"github.com/wsxiaoys/terminal/color"
)

func aptlySnapshotDiff(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 2 {
		cmd.Usage()
		return err
	}

	onlyMatching := cmd.Flag.Lookup("only-matching").Value.Get().(bool)

	snapshotCollection := debian.NewSnapshotCollection(context.database)
	packageCollection := debian.NewPackageCollection(context.database)

	// Load <name-a> snapshot
	snapshotA, err := snapshotCollection.ByName(args[0])
	if err != nil {
		return fmt.Errorf("unable to load snapshot A: %s", err)
	}

	err = snapshotCollection.LoadComplete(snapshotA)
	if err != nil {
		return fmt.Errorf("unable to load snapshot A: %s", err)
	}

	// Load <name-b> snapshot
	snapshotB, err := snapshotCollection.ByName(args[1])
	if err != nil {
		return fmt.Errorf("unable to load snapshot B: %s", err)
	}

	err = snapshotCollection.LoadComplete(snapshotB)
	if err != nil {
		return fmt.Errorf("unable to load snapshot B: %s", err)
	}

	// Calculate diff
	diff, err := snapshotA.RefList().Diff(snapshotB.RefList(), packageCollection)
	if err != nil {
		return fmt.Errorf("unable to calculate diff: %s", err)
	}

	if len(diff) == 0 {
		fmt.Printf("Snapshots are identical.\n")
	} else {
		fmt.Printf("  Arch   | Package                                  | Version in A                             | Version in B\n")
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

			color.Printf(code+" %-6s | %-40s | %-40s | %-40s\n", arch, pkg, verA, verB)
		}
	}

	return err
}

func makeCmdSnapshotDiff() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlySnapshotDiff,
		UsageLine: "diff <name-a> <name-b>",
		Short:     "calculates difference in packages between two snapshots",
		Long: `
Command diff shows list of missing and new packages, difference in package versions between two snapshots.

ex.
    $ aptly snapshot diff -only-matching wheezy-main wheezy-backports
`,
		Flag: *flag.NewFlagSet("aptly-snapshot-diff", flag.ExitOnError),
	}

	cmd.Flag.Bool("only-matching", false, "display diff only for matching packages (don't display missing packages)")

	return cmd
}
