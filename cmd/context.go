package cmd

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/console"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/files"
	"github.com/smira/aptly/http"
	"github.com/smira/aptly/utils"
	"github.com/smira/flag"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

// Common context shared by all commands
var context struct {
	progress          aptly.Progress
	downloader        aptly.Downloader
	database          database.Storage
	packagePool       aptly.PackagePool
	publishedStorage  aptly.PublishedStorage
	collectionFactory *debian.CollectionFactory
	dependencyOptions int
	architecturesList []string
	flags             *flag.FlagSet
	// Debug features
	fileCPUProfile *os.File
	fileMemProfile *os.File
	fileMemStats   *os.File
}

// InitContext initializes context with default settings
func InitContext(flags *flag.FlagSet) error {
	var err error

	context.flags = flags

	context.dependencyOptions = 0
	if utils.Config.DepFollowSuggests || flags.Lookup("dep-follow-suggests").Value.Get().(bool) {
		context.dependencyOptions |= debian.DepFollowSuggests
	}
	if utils.Config.DepFollowRecommends || flags.Lookup("dep-follow-recommends").Value.Get().(bool) {
		context.dependencyOptions |= debian.DepFollowRecommends
	}
	if utils.Config.DepFollowAllVariants || flags.Lookup("dep-follow-all-variants").Value.Get().(bool) {
		context.dependencyOptions |= debian.DepFollowAllVariants
	}
	if utils.Config.DepFollowSource || flags.Lookup("dep-follow-source").Value.Get().(bool) {
		context.dependencyOptions |= debian.DepFollowSource
	}

	context.architecturesList = utils.Config.Architectures
	optionArchitectures := flags.Lookup("architectures").Value.String()
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

	context.collectionFactory = debian.NewCollectionFactory(context.database)

	context.packagePool = files.NewPackagePool(utils.Config.RootDir)
	context.publishedStorage = files.NewPublishedStorage(utils.Config.RootDir)

	if aptly.EnableDebug {
		cpuprofile := flags.Lookup("cpuprofile").Value.String()
		if cpuprofile != "" {
			context.fileCPUProfile, err = os.Create(cpuprofile)
			if err != nil {
				return err
			}
			pprof.StartCPUProfile(context.fileCPUProfile)
		}

		memprofile := flags.Lookup("memprofile").Value.String()
		if memprofile != "" {
			context.fileMemProfile, err = os.Create(memprofile)
			if err != nil {
				return err
			}
		}

		memstats := flags.Lookup("memstats").Value.String()
		if memstats != "" {
			interval := flags.Lookup("meminterval").Value.Get().(time.Duration)

			context.fileMemStats, err = os.Create(memstats)
			if err != nil {
				return err
			}

			context.fileMemStats.WriteString("# Time\tHeapSys\tHeapAlloc\tHeapIdle\tHeapReleased\n")

			go func() {
				var stats runtime.MemStats

				start := time.Now().UnixNano()

				for {
					runtime.ReadMemStats(&stats)
					if context.fileMemStats != nil {
						context.fileMemStats.WriteString(fmt.Sprintf("%d\t%d\t%d\t%d\t%d\n",
							(time.Now().UnixNano()-start)/1000000, stats.HeapSys, stats.HeapAlloc, stats.HeapIdle, stats.HeapReleased))
						time.Sleep(interval)
					} else {
						break
					}
				}
			}()
		}
	}

	return nil
}

// ShutdownContext shuts context down
func ShutdownContext() {
	if aptly.EnableDebug {
		if context.fileMemProfile != nil {
			pprof.WriteHeapProfile(context.fileMemProfile)
			context.fileMemProfile.Close()
			context.fileMemProfile = nil
		}
		if context.fileCPUProfile != nil {
			pprof.StopCPUProfile()
			context.fileCPUProfile.Close()
			context.fileCPUProfile = nil
		}
		if context.fileMemProfile != nil {
			context.fileMemProfile.Close()
			context.fileMemProfile = nil
		}
	}
	if context.database != nil {
		context.database.Close()
	}
	if context.downloader != nil {
		context.downloader.Shutdown()
	}
	if context.progress != nil {
		context.progress.Shutdown()
	}
}
