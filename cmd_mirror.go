package main

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"log"
	"strings"
)

func aptlyMirrorList(cmd *commander.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		return
	}

	fmt.Printf("List of mirrors:\n")

	repoCollection := debian.NewRemoteRepoCollection(context.database)
	repoCollection.ForEach(func(repo *debian.RemoteRepo) {
		fmt.Printf(" * %s\n", repo)
	})

	fmt.Printf("\nTo get more information about repository, run `aptly mirror show <name>`.\n")
}

func aptlyMirrorCreate(cmd *commander.Command, args []string) {
	if len(args) < 3 {
		cmd.Usage()
		return
	}

	var architectures []string
	archs := cmd.Flag.Lookup("architecture").Value.String()
	if len(archs) > 0 {
		architectures = strings.Split(archs, ",")
	}

	repo, err := debian.NewRemoteRepo(args[0], args[1], args[2], args[3:], architectures)
	if err != nil {
		log.Fatalf("Unable to create mirror: %s", err)
	}

	err = repo.Fetch(context.downloader)
	if err != nil {
		log.Fatalf("Unable to fetch mirror: %s", err)
	}

	repoCollection := debian.NewRemoteRepoCollection(context.database)

	err = repoCollection.Add(repo)
	if err != nil {
		log.Fatalf("Unable to add mirror: %s", err)
	}

	fmt.Printf("\nMirror %s successfully added.\nYou can run 'aptly mirror update %s' to download repository contents.\n", repo, repo.Name)
}

func aptlyMirrorShow(cmd *commander.Command, args []string) {
	if len(args) != 1 {
		cmd.Usage()
		return
	}

	name := args[0]

	repoCollection := debian.NewRemoteRepoCollection(context.database)
	repo, err := repoCollection.ByName(name)
	if err != nil {
		log.Fatalf("Unable to show: %s", err)
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
	}
	if repo.PackageRefs != nil {
		fmt.Printf("Number of packages: %d\n", repo.PackageRefs.Len())
	}

	fmt.Printf("\nInformation from release file:\n")
	for name, value := range repo.Meta {
		fmt.Printf("%s: %s\n", name, value)
	}
}

func aptlyMirrorUpdate(cmd *commander.Command, args []string) {
	if len(args) != 1 {
		cmd.Usage()
		return
	}

	name := args[0]

	repoCollection := debian.NewRemoteRepoCollection(context.database)
	repo, err := repoCollection.ByName(name)
	if err != nil {
		log.Fatalf("Unable to update: %s", err)
	}

	err = repo.Fetch(context.downloader)
	if err != nil {
		log.Fatalf("Unable to update: %s", err)
	}

	err = repo.Download(context.downloader, context.database, context.packageRepository)
	if err != nil {
		log.Fatalf("Unable to update: %s", err)
	}

	err = repoCollection.Update(repo)
	if err != nil {
		log.Fatalf("Unable to update: %s", err)
	}

}

func makeCmdMirrorCreate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorCreate,
		UsageLine: "create",
		Short:     "create new mirror of Debian repository",
		Long: `
create only stores metadata about new mirror, and fetches Release files (it doesn't download packages)

ex:
  $ aptly mirror create <name> <archive url> <distribution> [<component1> ...]
`,
		Flag: *flag.NewFlagSet("aptly-mirror-create", flag.ExitOnError),
	}
	cmd.Flag.String("architecture", "", "limit architectures to specified in the list, comma-delimited list")

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
		UsageLine: "show",
		Short:     "show details about remote repository mirror",
		Long: `
show shows full information about mirror.

ex:
  $ aptly mirror show <name>
`,
		Flag: *flag.NewFlagSet("aptly-mirror-show", flag.ExitOnError),
	}

	return cmd
}

func makeCmdMirrorUpdate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorUpdate,
		UsageLine: "update",
		Short:     "update packages from remote mirror",
		Long: `
Update downloads list of packages and packages themselves.

ex:
  $ aptly mirror update <name>
`,
		Flag: *flag.NewFlagSet("aptly-mirror-update", flag.ExitOnError),
	}

	return cmd
}

func makeCmdMirror() *commander.Commander {
	return &commander.Commander{
		Name:  "mirror",
		Short: "manage mirrors of remote repositories",
		Commands: []*commander.Command{
			makeCmdMirrorCreate(),
			makeCmdMirrorList(),
			makeCmdMirrorShow(),
			//makeCmdMirrorDeestroy(),
			makeCmdMirrorUpdate(),
		},
		Flag: flag.NewFlagSet("aptly-mirror", flag.ExitOnError),
	}
}
