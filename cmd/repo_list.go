package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"sort"
)

func aptlyRepoList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	raw := cmd.Flag.Lookup("raw").Value.Get().(bool)

	repos := make([]string, context.CollectionFactory().LocalRepoCollection().Len())
	i := 0
	context.CollectionFactory().LocalRepoCollection().ForEach(func(repo *deb.LocalRepo) error {
		if raw {
			repos[i] = repo.Name
		} else {
			err := context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
			if err != nil {
				return err
			}

			repos[i] = fmt.Sprintf(" * %s (packages: %d)", repo.String(), repo.NumPackages())
		}
		i++
		return nil
	})

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

	cmd.Flag.Bool("raw", false, "display list in machine-readable format")

	return cmd
}
