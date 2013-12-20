package main

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
	"log"
	"os"
)

var cmd *commander.Command

func init() {
	cmd = &commander.Command{
		UsageLine: os.Args[0],
		Flag:      *flag.NewFlagSet("aptly", flag.ExitOnError),
		Subcommands: []*commander.Command{
			makeCmdMirror(),
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

	context.downloader = utils.NewDownloader(2)
	defer context.downloader.Shutdown()

	// TODO: configure DB dir
	context.database, err = database.OpenDB("/tmp/aptly/db")
	if err != nil {
		log.Fatalf("can't open database: %s", err)
	}
	defer context.database.Close()

	// TODO:configure pool dir
	context.packageRepository = debian.NewRepository("/tmp/aptly")

	err = cmd.Dispatch(os.Args[1:])
	if err != nil {
		log.Fatalf("%s", err)
	}
}
