package cmd

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func makeCmdSnapshot() *commander.Command {
	return &commander.Command{
		UsageLine: "snapshot",
		Short:     "manage snapshots of repositories",
		Subcommands: []*commander.Command{
			makeCmdSnapshotCreate(),
			makeCmdSnapshotList(),
			makeCmdSnapshotShow(),
			makeCmdSnapshotVerify(),
			makeCmdSnapshotPull(),
			makeCmdSnapshotDiff(),
			makeCmdSnapshotMerge(),
			makeCmdSnapshotDrop(),
		},
		Flag: *flag.NewFlagSet("aptly-snapshot", flag.ExitOnError),
	}
}
