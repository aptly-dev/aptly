package cmd

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"github.com/smira/commander"
	"sort"
)

func aptlyMirrorList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	raw := cmd.Flag.Lookup("raw").Value.Get().(bool)

	repos := make([]string, context.CollectionFactory().RemoteRepoCollection().Len())
	i := 0
	context.CollectionFactory().RemoteRepoCollection().ForEach(func(repo *deb.RemoteRepo) error {
		if raw {
			repos[i] = repo.Name
		} else {
			repos[i] = repo.String()
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

	cmd.Flag.Bool("raw", false, "display list in machine-readable format")

	return cmd
}
