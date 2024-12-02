package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/smira/commander"
	"gopkg.in/yaml.v3"
)

func aptlyConfigShow(_ *commander.Command, _ []string) error {
	show_yaml := context.Flags().Lookup("yaml").Value.Get().(bool)

	config := context.Config()

	if show_yaml {
		yamlData, err := yaml.Marshal(&config)
		if err != nil {
			return fmt.Errorf("error marshaling to YAML: %s", err)
		}

		fmt.Println(string(yamlData))
	} else {
		prettyJSON, err := json.MarshalIndent(config, "", "    ")
		if err != nil {
			return fmt.Errorf("unable to dump the config file: %s", err)
		}

		fmt.Println(string(prettyJSON))
	}

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
	cmd.Flag.Bool("yaml", false, "show yaml config")
	return cmd
}
