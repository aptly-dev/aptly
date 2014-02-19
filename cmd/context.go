package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/console"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/files"
	"github.com/smira/aptly/http"
	"github.com/smira/aptly/utils"
	"path/filepath"
	"strings"
)

// Common context shared by all commands
var context struct {
	progress          aptly.Progress
	downloader        aptly.Downloader
	database          database.Storage
	packagePool       aptly.PackagePool
	publishedStorage  aptly.PublishedStorage
	dependencyOptions int
	architecturesList []string
}

// InitContext initializes context with default settings
func InitContext(cmd *commander.Command) error {
	var err error

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
	if utils.Config.DepFollowSource || cmd.Flag.Lookup("dep-follow-source").Value.Get().(bool) {
		context.dependencyOptions |= debian.DepFollowSource
	}

	context.architecturesList = utils.Config.Architectures
	optionArchitectures := cmd.Flag.Lookup("architectures").Value.String()
	if optionArchitectures != "" {
		context.architecturesList = strings.Split(optionArchitectures, ",")
	}

	context.progress = console.NewProgress()
	context.progress.Start()

	context.downloader = http.NewDownloader(utils.Config.DownloadConcurrency, context.progress)

	context.database, err = database.OpenDB(filepath.Join(utils.Config.RootDir, "db"))
	if err != nil {
		return fmt.Errorf("can't open database: %s", err)
	}

	context.packagePool = files.NewPackagePool(utils.Config.RootDir)
	context.publishedStorage = files.NewPublishedStorage(utils.Config.RootDir)

	return nil
}

// ShutdownContext shuts context down
func ShutdownContext() {
	context.database.Close()
	context.downloader.Shutdown()
	context.progress.Shutdown()
}
