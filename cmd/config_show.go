package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/smira/commander"
)

func aptlyConfigShow(_ *commander.Command, _ []string) error {

	config := context.Config()
	prettyJSON, err := json.MarshalIndent(config, "", "    ")

	if err != nil {
		return fmt.Errorf("unable to dump the config file: %s", err)
	}

	fmt.Println(string(prettyJSON))

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
