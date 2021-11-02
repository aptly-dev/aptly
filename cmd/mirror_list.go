package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
)

func aptlyMirrorList(cmd *commander.Command, args []string) error {
	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	jsonFlag := cmd.Flag.Lookup("json").Value.Get().(bool)

	if jsonFlag {
		return aptlyMirrorListJson(cmd, args)
	}

	return aptlyMirrorListTxt(cmd, args)
}

func aptlyMirrorListTxt(cmd *commander.Command, args []string) error {
	var err error

	raw := cmd.Flag.Lookup("raw").Value.Get().(bool)
	collectionFactory := context.NewCollectionFactory()

	repos := make([]string, collectionFactory.RemoteRepoCollection().Len())
	i := 0
	collectionFactory.RemoteRepoCollection().ForEach(func(repo *deb.RemoteRepo) error {
		if raw {
			repos[i] = repo.Name
		} else {
			repos[i] = repo.String()
		}
		i++
		return nil
	})

	context.CloseDatabase()

	sort.Strings(repos)

	if raw {
		for _, repo := range repos {
			fmt.Printf("%s\n", repo)
		}
	} else {
		if len(repos) > 0 {
			fmt.Printf("List of mirrors:\n")
			for _, repo := range repos {
				fmt.Printf(" * %s\n", repo)
			}

			fmt.Printf("\nTo get more information about mirror, run `aptly mirror show <name>`.\n")
		} else {
			fmt.Printf("No mirrors found, create one with `aptly mirror create ...`.\n")
		}
	}
	return err
}

func aptlyMirrorListJson(cmd *commander.Command, args []string) error {
	var err error

	repos := make([]*deb.RemoteRepo, context.NewCollectionFactory().RemoteRepoCollection().Len())
	i := 0
	context.NewCollectionFactory().RemoteRepoCollection().ForEach(func(repo *deb.RemoteRepo) error {
		repos[i] = repo
		i++
		return nil
	})

	context.CloseDatabase()

	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	if output, e := json.MarshalIndent(repos, "", "  "); e == nil {
		fmt.Println(string(output))
	} else {
		err = e
	}

	return err
}

func makeCmdMirrorList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorList,
		UsageLine: "list",
		Short:     "list mirrors",
		Long: `
List shows full list of remote repository mirrors.

Example:

  $ aptly mirror list
`,
	}

	cmd.Flag.Bool("json", false, "display list in JSON format")
	cmd.Flag.Bool("raw", false, "display list in machine-readable format")

	return cmd
}
