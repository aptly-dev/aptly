package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/smira/commander"
)

func aptlyConfigShow(cmd *commander.Command, args []string) error {

	config := context.Config()
	pretty_json, err := json.MarshalIndent(config, "", "    ")

	if err != nil {
		return fmt.Errorf("unable to parse the config file: %s", err)
	}

	config_to_string := string(pretty_json)

	fmt.Println(config_to_string)

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
