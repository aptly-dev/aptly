package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
	"github.com/smira/aptly/utils"
)

func aptlyMirrorCreate(cmd *commander.Command, args []string) error {
	var err error
	if len(args) < 3 {
		cmd.Usage()
		return err
	}

	downloadSources := utils.Config.DownloadSourcePackages || cmd.Flag.Lookup("with-sources").Value.Get().(bool)

	repo, err := debian.NewRemoteRepo(args[0], args[1], args[2], args[3:], context.architecturesList, downloadSources)
	if err != nil {
		return fmt.Errorf("unable to create mirror: %s", err)
	}

	verifier, err := getVerifier(cmd)
	if err != nil {
		return fmt.Errorf("unable to initialize GPG verifier: %s", err)
	}

	err = repo.Fetch(context.downloader, verifier)
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

func makeCmdMirrorCreate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorCreate,
		UsageLine: "create <name> <archive url> <distribution> [<component1> ...]",
		Short:     "create new mirror of Debian repository",
		Long: `
Create records information about new mirror and fetches Release file (it doesn't download packages).

ex:
  $ aptly mirror create wheezy-main http://mirror.yandex.ru/debian/ wheezy main
`,
		Flag: *flag.NewFlagSet("aptly-mirror-create", flag.ExitOnError),
	}

	cmd.Flag.Bool("ignore-signatures", false, "disable verification of Release file signatures")
	cmd.Flag.Bool("with-sources", false, "download source packages")
	cmd.Flag.Var(&keyRings, "keyring", "gpg keyring to use when verifying Release file (could be specified multiple times)")

	return cmd
}
