package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"sort"
)

func aptlyRepoList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	if context.collectionFactory.LocalRepoCollection().Len() > 0 {
		fmt.Printf("List of mirrors:\n")
		repos := make([]string, context.collectionFactory.LocalRepoCollection().Len())
		i := 0
		context.collectionFactory.LocalRepoCollection().ForEach(func(repo *debian.LocalRepo) error {
			err := context.collectionFactory.LocalRepoCollection().LoadComplete(repo)
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
		Flag: *flag.NewFlagSet("aptly-repo-list", flag.ExitOnError),
	}

	return cmd
}
