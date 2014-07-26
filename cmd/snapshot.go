package cmd

import (
	"github.com/smira/commander"
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
			makeCmdSnapshotRename(),
		},
	}
}
