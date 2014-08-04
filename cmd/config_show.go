package cmd

import "fmt"
import "github.com/smira/commander"

func aptlyConfigShow(cmd *commander.Command, args []string) error {

	config := context.Config()

	fmt.Printf("RootDir: %s")

	return nil

}

func makeCmdConfigShow() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyConfigShow,
		UsageLine: "show",
		Short:     "show current aptly's config",
		Long: `
Command show displays the current aptly configuration.

Example:

  $ aptly config show

`,
	}
	return cmd
}
