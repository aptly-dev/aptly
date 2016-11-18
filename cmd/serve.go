package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/aptly-dev/aptly/aptly"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyServe(cmd *commander.Command, args []string) error {
	var err error

	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	// There are only two working options for aptly's rootDir:
	//   1. rootDir does not exist, then we'll create it
	//   2. rootDir exists and is writable
	// anything else must fail.
	// E.g.: Running the service under a different user may lead to a rootDir
	// that exists but is not usable due to access permissions.
	err = utils.DirIsAccessible(context.Config().RootDir)
	if err != nil {
		return err
	}

	collectionFactory := context.NewCollectionFactory()
	if collectionFactory.PublishedRepoCollection().Len() == 0 {
		fmt.Printf("No published repositories, unable to serve.\n")
		return nil
	}

	listen := context.Flags().Lookup("listen").Value.String()

	listenHost, listenPort, err := net.SplitHostPort(listen)

	if err != nil {
		return fmt.Errorf("wrong -listen specification: %s", err)
	}

	if listenHost == "" {
		listenHost, err = os.Hostname()
		if err != nil {
			listenHost = "localhost"
		}
	}

	fmt.Printf("Serving published repositories, recommended apt sources list:\n\n")

	sources := make(sort.StringSlice, 0, collectionFactory.PublishedRepoCollection().Len())
	published := make(map[string]*deb.PublishedRepo, collectionFactory.PublishedRepoCollection().Len())

	err = collectionFactory.PublishedRepoCollection().ForEach(func(repo *deb.PublishedRepo) error {
		e := collectionFactory.PublishedRepoCollection().LoadComplete(repo, collectionFactory)
		if e != nil {
			return e
		}

		sources = append(sources, repo.String())
		published[repo.String()] = repo

		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to serve: %s", err)
	}

	sort.Strings(sources)

	for _, source := range sources {
		repo := published[source]

		prefix := repo.Prefix
		if prefix == "." {
			prefix = ""
		} else {
			prefix += "/"
		}

		fmt.Printf("# %s\ndeb http://%s:%s/%s %s %s\n",
			repo, listenHost, listenPort, prefix, repo.Distribution, strings.Join(repo.Components(), " "))

		if utils.StrSliceHasItem(repo.Architectures, deb.ArchitectureSource) {
			fmt.Printf("deb-src http://%s:%s/%s %s %s\n",
				listenHost, listenPort, prefix, repo.Distribution, strings.Join(repo.Components(), " "))
		}
	}

	publicPath := context.GetPublishedStorage("").(aptly.FileSystemPublishedStorage).PublicPath()
	ShutdownContext()

	fmt.Printf("\nStarting web server at: %s (press Ctrl+C to quit)...\n", listen)

	err = http.ListenAndServe(listen, http.FileServer(http.Dir(publicPath)))
	if err != nil {
		return fmt.Errorf("unable to serve: %s", err)
	}
	return nil
}

func makeCmdServe() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyServe,
		UsageLine: "serve",
		Short:     "HTTP serve published repositories",
		Long: `
Command serve starts embedded HTTP server (not suitable for real production usage) to serve
contents of public/ subdirectory of aptly's root that contains published repositories.

Example:

  $ aptly serve -listen=:8080
`,
		Flag: *flag.NewFlagSet("aptly-serve", flag.ExitOnError),
	}

	cmd.Flag.String("listen", ":8080", "host:port for HTTP listening")

	return cmd
}
