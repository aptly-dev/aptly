package main

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
	"sort"
	"strings"
)

func aptlyMirrorList(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 0 {
		cmd.Usage()
		return err
	}

	repoCollection := debian.NewRemoteRepoCollection(context.database)

	if repoCollection.Len() > 0 {
		fmt.Printf("List of mirrors:\n")
		repos := make(sort.StringSlice, repoCollection.Len())
		i := 0
		repoCollection.ForEach(func(repo *debian.RemoteRepo) error {
			repos[i] = repo.String()
			i++
			return nil
		})

		sort.Strings(repos)
		for _, repo := range repos {
			fmt.Printf(" * %s\n", repo)
		}

		fmt.Printf("\nTo get more information about mirror, run `aptly mirror show <name>`.\n")
	} else {
		fmt.Printf("No mirrors found, create one with `aptly mirror create ...`.\n")
	}
	return err
}

func aptlyMirrorCreate(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 3 {
		cmd.Usage()
		return err
	}

	repo, err := debian.NewRemoteRepo(args[0], args[1], args[2], args[3:], context.architecturesList)
	if err != nil {
		return fmt.Errorf("unable to create mirror: %s", err)
	}

	err = repo.Fetch(context.downloader)
	if err != nil {
		return fmt.Errorf("unable to fetch mirror: %s", err)
	}

	repoCollection := debian.NewRemoteRepoCollection(context.database)

	err = repoCollection.Add(repo)
	if err != nil {
		return fmt.Errorf("unable to add mirror: %s", err)
	}

	fmt.Printf("\nMirror %s successfully added.\nYou can run 'aptly mirror update %s' to download repository contents.\n", repo, repo.Name)
	return err
}

func aptlyMirrorShow(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return err
	}

	name := args[0]

	repoCollection := debian.NewRemoteRepoCollection(context.database)
	repo, err := repoCollection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	err = repoCollection.LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to show: %s", err)
	}

	fmt.Printf("Name: %s\n", repo.Name)
	fmt.Printf("Archive Root URL: %s\n", repo.ArchiveRoot)
	fmt.Printf("Distribution: %s\n", repo.Distribution)
	fmt.Printf("Components: %s\n", strings.Join(repo.Components, ", "))
	fmt.Printf("Architectures: %s\n", strings.Join(repo.Architectures, ", "))
	if repo.LastDownloadDate.IsZero() {
		fmt.Printf("Last update: never\n")
	} else {
		fmt.Printf("Last update: %s\n", repo.LastDownloadDate.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("Number of packages: %d\n", repo.NumPackages())
	}

	fmt.Printf("\nInformation from release file:\n")
	for _, k := range utils.StrMapSortedKeys(repo.Meta) {
		fmt.Printf("%s: %s\n", k, repo.Meta[k])
	}

	return err
}

func aptlyMirrorUpdate(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return err
	}

	name := args[0]

	repoCollection := debian.NewRemoteRepoCollection(context.database)
	repo, err := repoCollection.ByName(name)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = repoCollection.LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = repo.Fetch(context.downloader)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	packageCollection := debian.NewPackageCollection(context.database)

	err = repo.Download(context.downloader, packageCollection, context.packageRepository)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = repoCollection.Update(repo)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	fmt.Printf("\nMirror `%s` has been successfully updated.\n", repo.Name)
	return err
}

func makeCmdMirrorCreate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorCreate,
		UsageLine: "create <name> <archive url> <distribution> [<component1> ...]",
		Short:     "create new mirror of Debian repository",
		Long: `
Create only stores metadata about new mirror, and fetches Release files (it doesn't download packages)
`,
		Flag: *flag.NewFlagSet("aptly-mirror-create", flag.ExitOnError),
	}

	return cmd
}

func makeCmdMirrorList() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorList,
		UsageLine: "list",
		Short:     "list mirrors of remote repositories",
		Long: `
list shows full list of remote repositories.

ex:
  $ aptly mirror list
`,
		Flag: *flag.NewFlagSet("aptly-mirror-list", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose output")

	return cmd
}

func makeCmdMirrorShow() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorShow,
		UsageLine: "show <name>",
		Short:     "show details about remote repository mirror",
		Long: `
show shows full information about mirror.
`,
		Flag: *flag.NewFlagSet("aptly-mirror-show", flag.ExitOnError),
	}

	return cmd
}

func makeCmdMirrorUpdate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorUpdate,
		UsageLine: "update <name>",
		Short:     "update packages from remote mirror",
		Long: `
Update downloads list of packages and packages themselves.
`,
		Flag: *flag.NewFlagSet("aptly-mirror-update", flag.ExitOnError),
	}

	return cmd
}

func makeCmdMirror() *commander.Command {
	return &commander.Command{
		UsageLine: "mirror",
		Short:     "manage mirrors of remote repositories",
		Subcommands: []*commander.Command{
			makeCmdMirrorCreate(),
			makeCmdMirrorList(),
			makeCmdMirrorShow(),
			//makeCmdMirrorDestroy(),
			makeCmdMirrorUpdate(),
		},
		Flag: *flag.NewFlagSet("aptly-mirror", flag.ExitOnError),
	}
}
