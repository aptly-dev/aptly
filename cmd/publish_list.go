package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
)

func aptlyPublishList(cmd *commander.Command, args []string) error {
	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	jsonFlag := cmd.Flag.Lookup("json").Value.Get().(bool)

	if jsonFlag {
		return aptlyPublishListJSON(cmd, args)
	}

	return aptlyPublishListTxt(cmd, args)
}

func aptlyPublishListTxt(cmd *commander.Command, _ []string) error {
	var err error

	raw := cmd.Flag.Lookup("raw").Value.Get().(bool)

	collectionFactory := context.NewCollectionFactory()
	published := make([]string, 0, collectionFactory.PublishedRepoCollection().Len())

	err = collectionFactory.PublishedRepoCollection().ForEach(func(repo *deb.PublishedRepo) error {
		e := collectionFactory.PublishedRepoCollection().LoadComplete(repo, collectionFactory)
		if e != nil {
			fmt.Fprintf(os.Stderr, "Error found on one publish (prefix:%s / distribution:%s / component:%s\n)",
				repo.StoragePrefix(), repo.Distribution, repo.Components())
			return e
		}

		if raw {
			published = append(published, fmt.Sprintf("%s %s", repo.StoragePrefix(), repo.Distribution))
		} else {
			published = append(published, repo.String())
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to load list of repos: %s", err)
	}

	context.CloseDatabase()

	sort.Strings(published)

	if raw {
		for _, info := range published {
			fmt.Printf("%s\n", info)
		}
	} else {
		if len(published) == 0 {
			fmt.Printf("No snapshots/local repos have been published. Publish a snapshot by running `aptly publish snapshot ...`.\n")
			return err
		}

		fmt.Printf("Published repositories:\n")

		for _, description := range published {
			fmt.Printf("  * %s\n", description)
		}
	}

	return err
}

func aptlyPublishListJSON(_ *commander.Command, _ []string) error {
	var err error

	repos := make([]*deb.PublishedRepo, 0, context.NewCollectionFactory().PublishedRepoCollection().Len())

	err = context.NewCollectionFactory().PublishedRepoCollection().ForEach(func(repo *deb.PublishedRepo) error {
		e := context.NewCollectionFactory().PublishedRepoCollection().LoadComplete(repo, context.NewCollectionFactory())
		if e != nil {
			fmt.Fprintf(os.Stderr, "Error found on one publish (prefix:%s / distribution:%s / component:%s\n)",
				repo.StoragePrefix(), repo.Distribution, repo.Components())
			return e
		}

		repos = append(repos, repo)

		return nil
	})

	if err != nil {
		return fmt.Errorf("unable to load list of repos: %s", err)
	}

	context.CloseDatabase()

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].GetPath() < repos[j].GetPath()
	})
	if output, e := json.MarshalIndent(repos, "", "  "); e == nil {
		fmt.Println(string(output))
	} else {
		err = e
	}

	return err
}

func makeCmdPublishList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishList,
		UsageLine: "list",
		Short:     "list of published repositories",
		Long: `
Display list of currently published snapshots.

Example:

    $ aptly publish list
`,
	}

	cmd.Flag.Bool("json", false, "display list in JSON format")
	cmd.Flag.Bool("raw", false, "display list in machine-readable format")

	return cmd
}
