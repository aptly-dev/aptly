package cmd

import (
	"github.com/smira/commander"
)

func makeCmdPackage() *commander.Command {
	return &commander.Command{
		UsageLine: "package",
		Short:     "operations on packages",
		Subcommands: []*commander.Command{
			makeCmdPackageSearch(),
			makeCmdPackageShow(),
		},
	}
}
