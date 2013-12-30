package main

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
	"log"
	"os"
	"path/filepath"
)

// aptly version
const Version = "0.1"

var cmd *commander.Command

func init() {
	cmd = &commander.Command{
		UsageLine: os.Args[0],
		Short:     "Debian repository management tool",
		Long: `
aptly allows to create partial and full mirrors of remote
repositories, filter them, merge, upgrade individual packages,
take snapshots and publish them back as Debian repositories.`,
		Flag: *flag.NewFlagSet("aptly", flag.ExitOnError),
		Subcommands: []*commander.Command{
			makeCmdMirror(),
			makeCmdSnapshot(),
			makeCmdPublish(),
			makeCmdVersion(),
		},
	}
}

var context struct {
	downloader        utils.Downloader
	database          database.Storage
	packageRepository *debian.Repository
}

func main() {
	err := cmd.Flag.Parse(os.Args[1:])
	if err != nil {
		log.Fatalf("%s", err)
	}

	configLocations := []string{
		filepath.Join(os.Getenv("HOME"), ".aptly.conf"),
		"/etc/aptly.conf",
	}

	for _, configLocation := range configLocations {
		err = utils.LoadConfig(configLocation, &utils.Config)
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Printf("Config file not found, creating default config at %s\n\n", configLocations[0])
		utils.SaveConfig(configLocations[0], &utils.Config)
	}

	context.downloader = utils.NewDownloader(utils.Config.DownloadConcurrency)
	defer context.downloader.Shutdown()

	// TODO: configure DB dir
	context.database, err = database.OpenDB(filepath.Join(utils.Config.RootDir, "db"))
	if err != nil {
		log.Fatalf("can't open database: %s", err)
	}
	defer context.database.Close()

	// TODO:configure pool dir
	context.packageRepository = debian.NewRepository(utils.Config.RootDir)

	err = cmd.Dispatch(os.Args[1:])
	if err != nil {
		log.Fatalf("%s", err)
	}
}
