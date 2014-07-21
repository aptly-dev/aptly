package cmd

import (
	"fmt"
	"github.com/smira/aptly/aptly"
	"github.com/smira/aptly/console"
	"github.com/smira/aptly/database"
	"github.com/smira/aptly/deb"
	"github.com/smira/aptly/files"
	"github.com/smira/aptly/http"
	"github.com/smira/aptly/s3"
	"github.com/smira/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"
)

// AptlyContext is a common context shared by all commands
type AptlyContext struct {
	flags        *flag.FlagSet
	configLoaded bool

	progress          aptly.Progress
	downloader        aptly.Downloader
	database          database.Storage
	packagePool       aptly.PackagePool
	publishedStorages map[string]aptly.PublishedStorage
	collectionFactory *deb.CollectionFactory
	dependencyOptions int
	architecturesList []string
	// Debug features
	fileCPUProfile *os.File
	fileMemProfile *os.File
	fileMemStats   *os.File
}

var context *AptlyContext

// Check interface
var _ aptly.PublishedStorageProvider = &AptlyContext{}

// FatalError is type for panicking to abort execution with non-zero
// exit code and print meaningful explanation
type FatalError struct {
	ReturnCode int
	Message    string
}

// Fatal panics and aborts execution with exit code 1
func Fatal(err error) {
	returnCode := 1
	if err == commander.ErrFlagError || err == commander.ErrCommandError {
		returnCode = 2
	}
	panic(&FatalError{ReturnCode: returnCode, Message: err.Error()})
}

// Config loads and returns current configuration
func (context *AptlyContext) Config() *utils.ConfigStructure {
	if !context.configLoaded {
		var err error

		configLocation := context.flags.Lookup("config").Value.String()
		if configLocation != "" {
			err = utils.LoadConfig(configLocation, &utils.Config)

			if err != nil {
				Fatal(err)
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
					Fatal(fmt.Errorf("error loading config file %s: %s", configLocation, err))
				}
			}

			if err != nil {
				fmt.Printf("Config file not found, creating default config at %s\n\n", configLocations[0])
				utils.SaveConfig(configLocations[0], &utils.Config)
			}
		}

		context.configLoaded = true

	}
	return &utils.Config
}

// DependencyOptions calculates options related to dependecy handling
func (context *AptlyContext) DependencyOptions() int {
	if context.dependencyOptions == -1 {
		context.dependencyOptions = 0
		if context.Config().DepFollowSuggests || context.flags.Lookup("dep-follow-suggests").Value.Get().(bool) {
			context.dependencyOptions |= deb.DepFollowSuggests
		}
		if context.Config().DepFollowRecommends || context.flags.Lookup("dep-follow-recommends").Value.Get().(bool) {
			context.dependencyOptions |= deb.DepFollowRecommends
		}
		if context.Config().DepFollowAllVariants || context.flags.Lookup("dep-follow-all-variants").Value.Get().(bool) {
			context.dependencyOptions |= deb.DepFollowAllVariants
		}
		if context.Config().DepFollowSource || context.flags.Lookup("dep-follow-source").Value.Get().(bool) {
			context.dependencyOptions |= deb.DepFollowSource
		}
	}

	return context.dependencyOptions
}

// ArchitecturesList returns list of architectures fixed via command line or config
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

// Progress creates or returns Progress object
func (context *AptlyContext) Progress() aptly.Progress {
	if context.progress == nil {
		context.progress = console.NewProgress()
		context.progress.Start()
	}

	return context.progress
}

// Downloader returns instance of current downloader
func (context *AptlyContext) Downloader() aptly.Downloader {
	if context.downloader == nil {
		var downloadLimit int64
		limitFlag := context.flags.Lookup("download-limit")
		if limitFlag != nil {
			downloadLimit = limitFlag.Value.Get().(int64)
		}
		if downloadLimit == 0 {
			downloadLimit = context.Config().DownloadLimit
		}
		context.downloader = http.NewDownloader(context.Config().DownloadConcurrency,
			downloadLimit*1024, context.Progress())
	}

	return context.downloader
}

// DBPath builds path to database
func (context *AptlyContext) DBPath() string {
	return filepath.Join(context.Config().RootDir, "db")
}

// Database opens and returns current instance of database
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

// CollectionFactory builds factory producing all kinds of collections
func (context *AptlyContext) CollectionFactory() *deb.CollectionFactory {
	if context.collectionFactory == nil {
		db, err := context.Database()
		if err != nil {
			Fatal(err)
		}
		context.collectionFactory = deb.NewCollectionFactory(db)
	}

	return context.collectionFactory
}

// PackagePool returns instance of PackagePool
func (context *AptlyContext) PackagePool() aptly.PackagePool {
	if context.packagePool == nil {
		context.packagePool = files.NewPackagePool(context.Config().RootDir)
	}

	return context.packagePool
}

// PublishedStorage returns instance of PublishedStorage
func (context *AptlyContext) GetPublishedStorage(name string) aptly.PublishedStorage {
	publishedStorage, ok := context.publishedStorages[name]
	if !ok {
		if name == "" {
			publishedStorage = files.NewPublishedStorage(context.Config().RootDir)
		} else if strings.HasPrefix(name, "s3:") {
			params, ok := context.Config().S3PublishRoots[name[3:]]
			if !ok {
				Fatal(fmt.Errorf("published S3 storage %v not configured", name[3:]))
			}

			var err error
			publishedStorage, err = s3.NewPublishedStorage(params.AccessKeyID, params.SecretAccessKey,
				params.Region, params.Bucket, params.ACL, params.Prefix)
			if err != nil {
				Fatal(err)
			}
		} else {
			Fatal(fmt.Errorf("unknown published storage format: %v", name))
		}
		context.publishedStorages[name] = publishedStorage
	}

	return publishedStorage
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

	context = &AptlyContext{
		flags:             flags,
		dependencyOptions: -1,
		publishedStorages: map[string]aptly.PublishedStorage{},
	}

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
