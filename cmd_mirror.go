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
	fmt.Printf("MIRROR LIST\n")
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

	log.Printf("Mirror %s successfully added.\nYou can run 'aptly mirror update %s' to download repository contents.\n", repo, repo.Name)
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

func makeCmdMirror() *commander.Commander {
	return &commander.Commander{
		Name:  "mirror",
		Short: "manage mirrors of remote repositories",
		Commands: []*commander.Command{
			makeCmdMirrorCreate(),
			makeCmdMirrorList(),
			//makeCmdMirrorShow(),
			//makeCmdMirrorDelete(),
			//makeCmdMirrorUpdate(),
		},
		Flag: flag.NewFlagSet("aptly-mirror", flag.ExitOnError),
	}
}
