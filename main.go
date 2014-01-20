package main

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
	"os"
	"path/filepath"
	"strings"
)

// aptly version
const Version = "0.2"

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

	cmd.Flag.Bool("dep-follow-suggests", false, "when processing dependencies, follow Suggests")
	cmd.Flag.Bool("dep-follow-recommends", false, "when processing dependencies, follow Recommends")
	cmd.Flag.Bool("dep-follow-all-variants", false, "when processing dependencies, follow a & b if depdency is 'a|b'")
	cmd.Flag.String("architectures", "", "list of architectures to consider during (comma-separated), default to all available")
}

var context struct {
	downloader        utils.Downloader
	database          database.Storage
	packageRepository *debian.Repository
	dependencyOptions int
	architecturesList []string
}

func fatal(err error) {
	fmt.Printf("ERROR: %s\n", err)
	os.Exit(1)
}

func main() {
	err := cmd.Flag.Parse(os.Args[1:])
	if err != nil {
		fatal(err)
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
		if !os.IsNotExist(err) {
			fatal(fmt.Errorf("error loading config file %s: %s", configLocation, err))
		}
	}

	if err != nil {
		fmt.Printf("Config file not found, creating default config at %s\n\n", configLocations[0])
		utils.SaveConfig(configLocations[0], &utils.Config)
	}

	context.dependencyOptions = 0
	if utils.Config.DepFollowSuggests || cmd.Flag.Lookup("dep-follow-suggests").Value.Get().(bool) {
		context.dependencyOptions |= debian.DepFollowSuggests
	}
	if utils.Config.DepFollowRecommends || cmd.Flag.Lookup("dep-follow-recommends").Value.Get().(bool) {
		context.dependencyOptions |= debian.DepFollowRecommends
	}
	if utils.Config.DepFollowAllVariants || cmd.Flag.Lookup("dep-follow-all-variants").Value.Get().(bool) {
		context.dependencyOptions |= debian.DepFollowAllVariants
	}

	context.architecturesList = utils.Config.Architectures
	optionArchitectures := cmd.Flag.Lookup("architectures").Value.String()
	if optionArchitectures != "" {
		context.architecturesList = strings.Split(optionArchitectures, ",")
	}

	context.downloader = utils.NewDownloader(utils.Config.DownloadConcurrency)
	defer context.downloader.Shutdown()

	context.database, err = database.OpenDB(filepath.Join(utils.Config.RootDir, "db"))
	if err != nil {
		fatal(fmt.Errorf("can't open database: %s", err))
	}
	defer context.database.Close()

	context.packageRepository = debian.NewRepository(utils.Config.RootDir)

	err = cmd.Dispatch(cmd.Flag.Args())
	if err != nil {
		fatal(err)
	}
}
