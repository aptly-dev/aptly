// Package context provides single entry to all resources
package context

import (
	gocontext "context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/console"
	"github.com/aptly-dev/aptly/database"
	"github.com/aptly-dev/aptly/database/goleveldb"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/files"
	"github.com/aptly-dev/aptly/http"
	"github.com/aptly-dev/aptly/pgp"
	"github.com/aptly-dev/aptly/s3"
	"github.com/aptly-dev/aptly/swift"
	"github.com/aptly-dev/aptly/task"
	"github.com/aptly-dev/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

// AptlyContext is a common context shared by all commands
type AptlyContext struct {
	sync.Mutex

	gocontext.Context

	flags, globalFlags *flag.FlagSet
	configLoaded       bool

	progress          aptly.Progress
	downloader        aptly.Downloader
	taskList          *task.List
	database          database.Storage
	packagePool       aptly.PackagePool
	publishedStorages map[string]aptly.PublishedStorage
	dependencyOptions int
	architecturesList []string
	// Debug features
	fileCPUProfile *os.File
	fileMemProfile *os.File
	fileMemStats   *os.File
}

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
	context.Lock()
	defer context.Unlock()

	return context.config()
}

func (context *AptlyContext) config() *utils.ConfigStructure {
	if !context.configLoaded {
		var err error

		configLocation := context.globalFlags.Lookup("config").Value.String()
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
				fmt.Fprintf(os.Stderr, "Config file not found, creating default config at %s\n\n", configLocations[0])

				// as this is fresh aptly installation, we don't need to support legacy pool locations
				utils.Config.SkipLegacyPool = true
				utils.SaveConfig(configLocations[0], &utils.Config)
			}
		}

		context.configLoaded = true

	}
	return &utils.Config
}

// LookupOption checks boolean flag with default (usually config) and command-line
// setting
func (context *AptlyContext) LookupOption(defaultValue bool, name string) (result bool) {
	context.Lock()
	defer context.Unlock()

	return context.lookupOption(defaultValue, name)
}

func (context *AptlyContext) lookupOption(defaultValue bool, name string) (result bool) {
	result = defaultValue

	if context.globalFlags.IsSet(name) {
		result = context.globalFlags.Lookup(name).Value.Get().(bool)
	}

	return
}

// DependencyOptions calculates options related to dependecy handling
func (context *AptlyContext) DependencyOptions() int {
	context.Lock()
	defer context.Unlock()

	if context.dependencyOptions == -1 {
		context.dependencyOptions = 0
		if context.lookupOption(context.config().DepFollowSuggests, "dep-follow-suggests") {
			context.dependencyOptions |= deb.DepFollowSuggests
		}
		if context.lookupOption(context.config().DepFollowRecommends, "dep-follow-recommends") {
			context.dependencyOptions |= deb.DepFollowRecommends
		}
		if context.lookupOption(context.config().DepFollowAllVariants, "dep-follow-all-variants") {
			context.dependencyOptions |= deb.DepFollowAllVariants
		}
		if context.lookupOption(context.config().DepFollowSource, "dep-follow-source") {
			context.dependencyOptions |= deb.DepFollowSource
		}
		if context.lookupOption(context.config().DepVerboseResolve, "dep-verbose-resolve") {
			context.dependencyOptions |= deb.DepVerboseResolve
		}
	}

	return context.dependencyOptions
}

// ArchitecturesList returns list of architectures fixed via command line or config
func (context *AptlyContext) ArchitecturesList() []string {
	context.Lock()
	defer context.Unlock()

	if context.architecturesList == nil {
		context.architecturesList = context.config().Architectures
		optionArchitectures := context.globalFlags.Lookup("architectures").Value.String()
		if optionArchitectures != "" {
			context.architecturesList = strings.Split(optionArchitectures, ",")
		}
	}

	return context.architecturesList
}

// Progress creates or returns Progress object
func (context *AptlyContext) Progress() aptly.Progress {
	context.Lock()
	defer context.Unlock()

	return context._progress()
}

func (context *AptlyContext) _progress() aptly.Progress {
	if context.progress == nil {
		context.progress = console.NewProgress()
		context.progress.Start()
	}

	return context.progress
}

// NewDownloader returns instance of new downloader with given progress
func (context *AptlyContext) NewDownloader(progress aptly.Progress) aptly.Downloader {
	context.Lock()
	defer context.Unlock()

	return context.newDownloader(progress)
}

// NewDownloader returns instance of new downloader with given progress without locking
// so it can be used for internal usage.
func (context *AptlyContext) newDownloader(progress aptly.Progress) aptly.Downloader {
	var downloadLimit int64
	limitFlag := context.flags.Lookup("download-limit")
	if limitFlag != nil {
		downloadLimit = limitFlag.Value.Get().(int64)
	}
	if downloadLimit == 0 {
		downloadLimit = context.config().DownloadLimit
	}
	maxTries := context.config().DownloadRetries + 1
	maxTriesFlag := context.flags.Lookup("max-tries")
	if maxTriesFlag != nil {
		// If flag is defined prefer it to global setting
		maxTries = maxTriesFlag.Value.Get().(int)
	}
	return http.NewDownloader(downloadLimit*1024, maxTries, progress)
}

// Downloader returns instance of current downloader
func (context *AptlyContext) Downloader() aptly.Downloader {
	context.Lock()
	defer context.Unlock()

	if context.downloader == nil {
		context.downloader = context.newDownloader(context._progress())
	}

	return context.downloader
}

// TaskList returns instance of current task list
func (context *AptlyContext) TaskList() *task.List {
	context.Lock()
	defer context.Unlock()

	if context.taskList == nil {
		context.taskList = task.NewList()
	}
	return context.taskList
}

// DBPath builds path to database
func (context *AptlyContext) DBPath() string {
	context.Lock()
	defer context.Unlock()

	return context.dbPath()
}

// DBPath builds path to database
func (context *AptlyContext) dbPath() string {
	return filepath.Join(context.config().RootDir, "db")
}

// Database opens and returns current instance of database
func (context *AptlyContext) Database() (database.Storage, error) {
	context.Lock()
	defer context.Unlock()

	return context._database()
}

func (context *AptlyContext) _database() (database.Storage, error) {
	if context.database == nil {
		var err error

		context.database, err = goleveldb.NewDB(context.dbPath())
		if err != nil {
			return nil, fmt.Errorf("can't instantiate database: %s", err)
		}
	}

	var tries int
	if context.config().DatabaseOpenAttempts == -1 {
		tries = context.flags.Lookup("db-open-attempts").Value.Get().(int)
	} else {
		tries = context.config().DatabaseOpenAttempts
	}

	const BaseDelay = 10 * time.Second
	const Jitter = 1 * time.Second

	for ; tries >= 0; tries-- {
		err := context.database.Open()
		if err == nil || !strings.Contains(err.Error(), "resource temporarily unavailable") {
			return context.database, err
		}

		if tries > 0 {
			delay := time.Duration(rand.NormFloat64()*float64(Jitter) + float64(BaseDelay))
			if delay < 0 {
				delay = time.Second
			}

			context._progress().PrintfStdErr("Unable to open database, sleeping %s, attempts left %d...\n", delay, tries)
			time.Sleep(delay)
		}
	}

	return nil, fmt.Errorf("unable to reopen the DB, maximum number of retries reached")
}

// CloseDatabase closes the db temporarily
func (context *AptlyContext) CloseDatabase() error {
	context.Lock()
	defer context.Unlock()

	if context.database == nil {
		return nil
	}

	return context.database.Close()
}

// ReOpenDatabase reopens the db after close
func (context *AptlyContext) ReOpenDatabase() error {
	_, err := context.Database()

	return err
}

// NewCollectionFactory builds factory producing all kinds of collections
func (context *AptlyContext) NewCollectionFactory() *deb.CollectionFactory {
	context.Lock()
	defer context.Unlock()

	db, err := context._database()
	if err != nil {
		Fatal(err)
	}
	return deb.NewCollectionFactory(db)
}

// PackagePool returns instance of PackagePool
func (context *AptlyContext) PackagePool() aptly.PackagePool {
	context.Lock()
	defer context.Unlock()

	if context.packagePool == nil {
		context.packagePool = files.NewPackagePool(context.config().RootDir, !context.config().SkipLegacyPool)
	}

	return context.packagePool
}

// GetPublishedStorage returns instance of PublishedStorage
func (context *AptlyContext) GetPublishedStorage(name string) aptly.PublishedStorage {
	context.Lock()
	defer context.Unlock()

	publishedStorage, ok := context.publishedStorages[name]
	if !ok {
		if name == "" {
			publishedStorage = files.NewPublishedStorage(filepath.Join(context.config().RootDir, "public"), "hardlink", "")
		} else if strings.HasPrefix(name, "filesystem:") {
			params, ok := context.config().FileSystemPublishRoots[name[11:]]
			if !ok {
				Fatal(fmt.Errorf("published local storage %v not configured", name[11:]))
			}

			publishedStorage = files.NewPublishedStorage(params.RootDir, params.LinkMethod, params.VerifyMethod)
		} else if strings.HasPrefix(name, "s3:") {
			params, ok := context.config().S3PublishRoots[name[3:]]
			if !ok {
				Fatal(fmt.Errorf("published S3 storage %v not configured", name[3:]))
			}

			var err error
			publishedStorage, err = s3.NewPublishedStorage(
				params.AccessKeyID, params.SecretAccessKey, params.SessionToken,
				params.Region, params.Endpoint, params.Bucket, params.ACL, params.Prefix, params.StorageClass,
				params.EncryptionMethod, params.PlusWorkaround, params.DisableMultiDel,
				params.ForceSigV2, params.Debug)
			if err != nil {
				Fatal(err)
			}
		} else if strings.HasPrefix(name, "swift:") {
			params, ok := context.config().SwiftPublishRoots[name[6:]]
			if !ok {
				Fatal(fmt.Errorf("published Swift storage %v not configured", name[6:]))
			}

			var err error
			publishedStorage, err = swift.NewPublishedStorage(params.UserName, params.Password,
				params.AuthURL, params.Tenant, params.TenantID, params.Domain, params.DomainID, params.TenantDomain, params.TenantDomainID, params.Container, params.Prefix)
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

// UploadPath builds path to upload storage
func (context *AptlyContext) UploadPath() string {
	return filepath.Join(context.Config().RootDir, "upload")
}

func (context *AptlyContext) pgpProvider() string {
	var provider string

	if context.globalFlags.IsSet("gpg-provider") {
		provider = context.globalFlags.Lookup("gpg-provider").Value.String()
	} else {
		provider = context.config().GpgProvider
	}

	switch provider {
	case "gpg": // nolint: goconst
	case "gpg1": // nolint: goconst
	case "gpg2": // nolint: goconst
	case "internal": // nolint: goconst
	default:
		Fatal(fmt.Errorf("unknown gpg provider: %v", provider))
	}

	return provider
}

func (context *AptlyContext) getGPGFinder(provider string) pgp.GPGFinder {
	switch context.pgpProvider() {
	case "gpg1":
		return pgp.GPG1Finder()
	case "gpg2":
		return pgp.GPG2Finder()
	case "gpg":
		return pgp.GPGDefaultFinder()
	}

	panic("uknown GPG provider type")
}

// GetSigner returns Signer with respect to provider
func (context *AptlyContext) GetSigner() pgp.Signer {
	context.Lock()
	defer context.Unlock()

	provider := context.pgpProvider()
	if provider == "internal" { // nolint: goconst
		return &pgp.GoSigner{}
	}

	return pgp.NewGpgSigner(context.getGPGFinder(provider))
}

// GetVerifier returns Verifier with respect to provider
func (context *AptlyContext) GetVerifier() pgp.Verifier {
	context.Lock()
	defer context.Unlock()

	provider := context.pgpProvider()
	if provider == "internal" { // nolint: goconst
		return &pgp.GoVerifier{}
	}

	return pgp.NewGpgVerifier(context.getGPGFinder(provider))
}

// UpdateFlags sets internal copy of flags in the context
func (context *AptlyContext) UpdateFlags(flags *flag.FlagSet) {
	context.Lock()
	defer context.Unlock()

	context.flags = flags
}

// Flags returns current command flags
func (context *AptlyContext) Flags() *flag.FlagSet {
	context.Lock()
	defer context.Unlock()

	return context.flags
}

// GlobalFlags returns flags passed to all commands
func (context *AptlyContext) GlobalFlags() *flag.FlagSet {
	context.Lock()
	defer context.Unlock()

	return context.globalFlags
}

// GoContextHandleSignals upgrades context to handle ^C by aborting context
func (context *AptlyContext) GoContextHandleSignals() {
	context.Lock()
	defer context.Unlock()

	// Catch ^C
	sigch := make(chan os.Signal)
	signal.Notify(sigch, os.Interrupt)

	var cancel gocontext.CancelFunc

	context.Context, cancel = gocontext.WithCancel(context.Context)

	go func() {
		<-sigch
		signal.Stop(sigch)
		context.Progress().PrintfStdErr("Aborting... press ^C once again to abort immediately\n")
		cancel()
	}()
}

// Shutdown shuts context down
func (context *AptlyContext) Shutdown() {
	context.Lock()
	defer context.Unlock()

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
		context.database = nil
	}
	if context.downloader != nil {
		context.downloader = nil
	}
	if context.progress != nil {
		context.progress.Shutdown()
		context.progress = nil
	}
}

// Cleanup does partial shutdown of context
func (context *AptlyContext) Cleanup() {
	context.Lock()
	defer context.Unlock()

	if context.downloader != nil {
		context.downloader = nil
	}
	if context.progress != nil {
		context.progress.Shutdown()
		context.progress = nil
	}
}

// NewContext initializes context with default settings
func NewContext(flags *flag.FlagSet) (*AptlyContext, error) {
	var err error

	context := &AptlyContext{
		flags:             flags,
		globalFlags:       flags,
		dependencyOptions: -1,
		Context:           gocontext.TODO(),
		publishedStorages: map[string]aptly.PublishedStorage{},
	}

	if aptly.EnableDebug {
		cpuprofile := flags.Lookup("cpuprofile").Value.String()
		if cpuprofile != "" {
			context.fileCPUProfile, err = os.Create(cpuprofile)
			if err != nil {
				return nil, err
			}
			pprof.StartCPUProfile(context.fileCPUProfile)
		}

		memprofile := flags.Lookup("memprofile").Value.String()
		if memprofile != "" {
			context.fileMemProfile, err = os.Create(memprofile)
			if err != nil {
				return nil, err
			}
		}

		memstats := flags.Lookup("memstats").Value.String()
		if memstats != "" {
			interval := flags.Lookup("meminterval").Value.Get().(time.Duration)

			context.fileMemStats, err = os.Create(memstats)
			if err != nil {
				return nil, err
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

	return context, nil
}
