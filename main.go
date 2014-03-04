package main

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/smira/aptly/cmd"
	"github.com/smira/aptly/utils"
	"os"
	"path/filepath"
)

var returnCode = 0

func fatal(err error) {
	fmt.Printf("ERROR: %s\n", err)
	returnCode = 1
}

func loadConfig(command *commander.Command) error {
	var err error

	configLocation := command.Flag.Lookup("config").Value.String()
	if configLocation != "" {
		err = utils.LoadConfig(configLocation, &utils.Config)

		if err != nil {
			return err
		}
	} else {
		configLocations := []string{
			filepath.Join(os.Getenv("HOME"), ".aptly.conf"),
			"/etc/aptly.conf",
		}

		for _, configLocation := range configLocations {
			err = utils.LoadConfig(configLocation, &utils.Config)
			if err == nil {
				break
			}
			if !os.IsNotExist(err) {
				fatal(fmt.Errorf("error loading config file %s: %s", configLocation, err))
				return nil
			}
		}

		if err != nil {
			fmt.Printf("Config file not found, creating default config at %s\n\n", configLocations[0])
			utils.SaveConfig(configLocations[0], &utils.Config)
		}
	}

	return nil
}

func main() {
	defer func() { os.Exit(returnCode) }()

	command := cmd.RootCommand()

	err := command.Flag.Parse(os.Args[1:])
	if err != nil {
		fatal(err)
		return
	}

	err = loadConfig(command)
	if err != nil {
		fatal(err)
		return
	}
	if returnCode != 0 {
		return
	}

	err = cmd.InitContext(command)
	if err != nil {
		fatal(err)
		return
	}
	defer cmd.ShutdownContext()

	err = command.Dispatch(command.Flag.Args())
	if err != nil {
		fatal(err)
		return
	}
}
