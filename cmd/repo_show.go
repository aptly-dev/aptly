package cmd

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/aptly-dev/aptly/deb"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyRepoShow(cmd *commander.Command, args []string) error {
	if len(args) != 1 {
		cmd.Usage()
		return commander.ErrCommandError
	}

	jsonFlag := cmd.Flag.Lookup("json").Value.Get().(bool)

	if jsonFlag {
		return aptlyRepoShowJson(cmd, args)
	}

	return aptlyRepoShowTxt(cmd, args)
}

func aptlyRepoShowTxt(cmd *commander.Command, args []string) error {
	var err error

	name := args[0]

	collectionFactory := context.NewCollectionFactory()
	repo, err := collectionFactory.LocalRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = collectionFactory.LocalRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	fmt.Printf("Name: %s\n", repo.Name)
	fmt.Printf("Comment: %s\n", repo.Comment)
	fmt.Printf("Default Distribution: %s\n", repo.DefaultDistribution)
	fmt.Printf("Default Component: %s\n", repo.DefaultComponent)
	if repo.Uploaders != nil {
		fmt.Printf("Uploaders: %s\n", repo.Uploaders)
	}
	fmt.Printf("Number of packages: %d\n", repo.NumPackages())

	withPackages := context.Flags().Lookup("with-packages").Value.Get().(bool)
	if withPackages {
		ListPackagesRefList(repo.RefList(), collectionFactory)
	}

	return err
}

func aptlyRepoShowJson(cmd *commander.Command, args []string) error {
	var err error

	name := args[0]

	repo, err := context.NewCollectionFactory().LocalRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = context.NewCollectionFactory().LocalRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	// include packages if requested
	packageList := []string{}
	withPackages := context.Flags().Lookup("with-packages").Value.Get().(bool)
	if withPackages {
		if repo.RefList() != nil {
			var list *deb.PackageList
			list, err = deb.NewPackageListFromRefList(repo.RefList(), context.NewCollectionFactory().PackageCollection(), context.Progress())
			if err == nil {
				packageList = list.FullNames()
			}
		}

		sort.Strings(packageList)
	}

	// merge the repo object with the package list
	var output []byte
	if output, err = json.MarshalIndent(struct {
		*deb.LocalRepo
		Packages []string
	}{repo, packageList}, "", "  "); err == nil {
		fmt.Println(string(output))
	}

	return err
}

func makeCmdRepoShow() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyRepoShow,
		UsageLine: "show <name>",
		Short:     "show details about local repository",
		Long: `
Show command shows full information about local package repository.

ex:
  $ aptly repo show testing
`,
		Flag: *flag.NewFlagSet("aptly-repo-show", flag.ExitOnError),
	}

	cmd.Flag.Bool("json", false, "display record in JSON format")
	cmd.Flag.Bool("with-packages", false, "show list of packages")

	return cmd
}
