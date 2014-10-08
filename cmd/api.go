package cmd

import (
	"github.com/smira/commander"
)

func makeCmdAPI() *commander.Command {
	return &commander.Command{
		UsageLine: "api",
		Short:     "start API server/issue requests",
		Subcommands: []*commander.Command{
			makeCmdAPIServe(),
		},
	}
}
