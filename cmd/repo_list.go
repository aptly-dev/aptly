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
		return err
	}

	if context.CollectionFactory().LocalRepoCollection().Len() > 0 {
		fmt.Printf("List of mirrors:\n")
		repos := make([]string, context.CollectionFactory().LocalRepoCollection().Len())
		i := 0
		context.CollectionFactory().LocalRepoCollection().ForEach(func(repo *deb.LocalRepo) error {
			err := context.CollectionFactory().LocalRepoCollection().LoadComplete(repo)
			if err != nil {
				return err
			}

			repos[i] = fmt.Sprintf(" * %s (packages: %d)", repo.String(), repo.NumPackages())
			i++
			return nil
		})

		sort.Strings(repos)
		for _, repo := range repos {
			fmt.Println(repo)
		}

		fmt.Printf("\nTo get more information about local repository, run `aptly repo show <name>`.\n")
	} else {
		fmt.Printf("No local repositories found, create one with `aptly repo create ...`.\n")
	}
	return err
}

func makeCmdRepoList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoList,
		UsageLine: "list",
		Short:     "list local repositories",
		Long: `
List shows full list of local package repositories.

Example:

  $ aptly repo list
`,
	}

	return cmd
}
