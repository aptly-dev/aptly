package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
)

func aptlyRepoList(cmd *commander.Command, args []string) error {
	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	jsonFlag := cmd.Flag.Lookup("json").Value.Get().(bool)

	if jsonFlag {
		return aptlyRepoListJson(cmd, args)
	}

	return aptlyRepoListTxt(cmd, args)
}

func aptlyRepoListTxt(cmd *commander.Command, args []string) error {
	var err error

	raw := cmd.Flag.Lookup("raw").Value.Get().(bool)

	collectionFactory := context.NewCollectionFactory()
	repos := make([]string, collectionFactory.LocalRepoCollection().Len())
	i := 0
	collectionFactory.LocalRepoCollection().ForEach(func(repo *deb.LocalRepo) error {
		if raw {
			repos[i] = repo.Name
		} else {
			e := collectionFactory.LocalRepoCollection().LoadComplete(repo)
			if e != nil {
				return e
			}

			repos[i] = fmt.Sprintf(" * %s (packages: %d)", repo.String(), repo.NumPackages())
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
			fmt.Printf("List of local repos:\n")
			for _, repo := range repos {
				fmt.Println(repo)
			}

			fmt.Printf("\nTo get more information about local repository, run `aptly repo show <name>`.\n")
		} else {
			fmt.Printf("No local repositories found, create one with `aptly repo create ...`.\n")
		}
	}

	return err
}

func aptlyRepoListJson(cmd *commander.Command, args []string) error {
	var err error

	repos := make([]*deb.LocalRepo, context.NewCollectionFactory().LocalRepoCollection().Len())
	i := 0
	context.NewCollectionFactory().LocalRepoCollection().ForEach(func(repo *deb.LocalRepo) error {
		e := context.NewCollectionFactory().LocalRepoCollection().LoadComplete(repo)
		if e != nil {
			return e
		}

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

func makeCmdRepoList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoList,
		UsageLine: "list",
		Short:     "list local repositories",
		Long: `
List command shows full list of local package repositories.

Example:

  $ aptly repo list
`,
	}

	cmd.Flag.Bool("json", false, "display list in JSON format")
	cmd.Flag.Bool("raw", false, "display list in machine-readable format")

	return cmd
}
