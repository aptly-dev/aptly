package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func ex_make_cmd_cmd2() *commander.Command {
	cmd := &commander.Command{
		Run:       ex_run_cmd_cmd2,
		UsageLine: "cmd2 [options]",
		Short:     "runs cmd2 and exits",
		Long: `
runs cmd2 and exits.

ex:
 $ my-cmd cmd2
`,
		Flag: *flag.NewFlagSet("my-cmd-cmd2", flag.ExitOnError),
	}
	cmd.Flag.Bool("q", true, "only print error and warning messages, all other output will be suppressed")
	return cmd
}

func ex_run_cmd_cmd2(cmd *commander.Command, args []string) error {
	name := "my-cmd-" + cmd.Name()
	quiet := cmd.Flag.Lookup("q").Value.Get().(bool)
	fmt.Printf("%s: hello from cmd2 (quiet=%v)\n", name, quiet)
	return nil
}

// EOF
