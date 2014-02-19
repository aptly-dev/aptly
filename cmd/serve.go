package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
	"net"
	"net/http"
	"os"
	"sort"
)

func aptlyServe(cmd *commander.Command, args []string) error {
	var err error

	publishedCollection := debian.NewPublishedRepoCollection(context.database)
	snapshotCollection := debian.NewSnapshotCollection(context.database)

	if publishedCollection.Len() == 0 {
		fmt.Printf("No published repositories, unable to serve.\n")
		return nil
	}

	listen := cmd.Flag.Lookup("listen").Value.String()

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

	sources := make(sort.StringSlice, 0, publishedCollection.Len())
	published := make(map[string]*debian.PublishedRepo, publishedCollection.Len())

	err = publishedCollection.ForEach(func(repo *debian.PublishedRepo) error {
		err := publishedCollection.LoadComplete(repo, snapshotCollection)
		if err != nil {
			return err
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
			repo, listenHost, listenPort, prefix, repo.Distribution, repo.Component)

		if utils.StrSliceHasItem(repo.Architectures, "source") {
			fmt.Printf("deb-src http://%s:%s/%s %s %s\n",
				listenHost, listenPort, prefix, repo.Distribution, repo.Component)
		}
	}

	context.database.Close()

	fmt.Printf("\nStarting web server at: %s (press Ctrl+C to quit)...\n", listen)

	err = http.ListenAndServe(listen, http.FileServer(http.Dir(context.publishedStorage.PublicPath())))
	if err != nil {
		return fmt.Errorf("unable to serve: %s", err)
	}
	return nil
}

func makeCmdServe() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyServe,
		UsageLine: "serve",
		Short:     "start embedded HTTP server to serve published repositories",
		Long: `
Command serve starts embedded HTTP server (not suitable for real production usage) to serve
contents of public/ subdirectory of aptly's root that contains published repositories.

ex:
  $ aptly serve -listen=:8080
`,
		Flag: *flag.NewFlagSet("aptly-serve", flag.ExitOnError),
	}

	cmd.Flag.String("listen", ":8080", "host:port for HTTP listening")

	return cmd
}
