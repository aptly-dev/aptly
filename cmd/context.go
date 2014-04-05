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
type AptlyContext struct {
	flags *flag.FlagSet

	progress          aptly.Progress
	downloader        aptly.Downloader
	database          database.Storage
	packagePool       aptly.PackagePool
	publishedStorage  aptly.PublishedStorage
	collectionFactory *debian.CollectionFactory
	dependencyOptions int
	architecturesList []string
	// Debug features
	fileCPUProfile *os.File
	fileMemProfile *os.File
	fileMemStats   *os.File
}

var context *AptlyContext

func (context *AptlyContext) Config() *utils.ConfigStructure {
	return &utils.Config
}

func (context *AptlyContext) DependencyOptions() int {
	if context.dependencyOptions == -1 {
		context.dependencyOptions = 0
		if context.Config().DepFollowSuggests || context.flags.Lookup("dep-follow-suggests").Value.Get().(bool) {
			context.dependencyOptions |= debian.DepFollowSuggests
		}
		if context.Config().DepFollowRecommends || context.flags.Lookup("dep-follow-recommends").Value.Get().(bool) {
			context.dependencyOptions |= debian.DepFollowRecommends
		}
		if context.Config().DepFollowAllVariants || context.flags.Lookup("dep-follow-all-variants").Value.Get().(bool) {
			context.dependencyOptions |= debian.DepFollowAllVariants
		}
		if context.Config().DepFollowSource || context.flags.Lookup("dep-follow-source").Value.Get().(bool) {
			context.dependencyOptions |= debian.DepFollowSource
		}
	}

	return context.dependencyOptions
}

func (context *AptlyContext) ArchitecturesList() []string {
	if context.architecturesList == nil {
		context.architecturesList = context.Config().Architectures
		optionArchitectures := context.flags.Lookup("architectures").Value.String()
		if optionArchitectures != "" {
			context.architecturesList = strings.Split(optionArchitectures, ",")
		}
	}

	return context.architecturesList
}

func (context *AptlyContext) Progress() aptly.Progress {
	if context.progress == nil {
		context.progress = console.NewProgress()
		context.progress.Start()
	}

	return context.progress
}

func (context *AptlyContext) Downloader() aptly.Downloader {
	if context.downloader == nil {
		context.downloader = http.NewDownloader(context.Config().DownloadConcurrency, context.Progress())
	}

	return context.downloader
}

func (context *AptlyContext) DBPath() string {
	return filepath.Join(context.Config().RootDir, "db")
}

func (context *AptlyContext) Database() (database.Storage, error) {
	if context.database == nil {
		var err error

		context.database, err = database.OpenDB(context.DBPath())
		if err != nil {
			return nil, fmt.Errorf("can't open database: %s", err)
		}
	}

	return context.database, nil
}

func (context *AptlyContext) CollectionFactory() *debian.CollectionFactory {
	if context.collectionFactory == nil {
		db, err := context.Database()
		if err != nil {
			panic(err)
		}
		context.collectionFactory = debian.NewCollectionFactory(db)
	}

	return context.collectionFactory
}

func (context *AptlyContext) PackagePool() aptly.PackagePool {
	if context.packagePool == nil {
		context.packagePool = files.NewPackagePool(context.Config().RootDir)
	}

	return context.packagePool
}

func (context *AptlyContext) PublishedStorage() aptly.PublishedStorage {
	if context.publishedStorage == nil {
		context.publishedStorage = files.NewPublishedStorage(context.Config().RootDir)
	}

	return context.publishedStorage
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

// InitContext initializes context with default settings
func InitContext(flags *flag.FlagSet) error {
	var err error

	context = &AptlyContext{flags: flags, dependencyOptions: -1}

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
