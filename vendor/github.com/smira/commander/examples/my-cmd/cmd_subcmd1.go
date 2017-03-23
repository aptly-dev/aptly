package main

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

var cmd_subcmd1 = &commander.Command{
	UsageLine: "subcmd1 <command>",
	Short:     "subcmd1 subcommand. does subcmd1 thingies",
	Subcommands: []*commander.Command{
		cmd_subcmd1_cmd1,
		cmd_subcmd1_cmd2,
	},
	Flag: *flag.NewFlagSet("my-cmd-subcmd1", flag.ExitOnError),
}

// EOF
