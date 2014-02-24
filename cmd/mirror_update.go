package cmd

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/debian"
)

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

	ignoreMismatch := cmd.Flag.Lookup("ignore-checksums").Value.Get().(bool)

	verifier, err := getVerifier(cmd)
	if err != nil {
		return fmt.Errorf("unable to initialize GPG verifier: %s", err)
	}

	err = repo.Fetch(context.downloader, verifier)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	packageCollection := debian.NewPackageCollection(context.database)

	err = repo.Download(context.progress, context.downloader, packageCollection, context.packagePool, ignoreMismatch)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = repoCollection.Update(repo)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	context.progress.Printf("\nMirror `%s` has been successfully updated.\n", repo.Name)
	return err
}

func makeCmdMirrorUpdate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorUpdate,
		UsageLine: "update <name>",
		Short:     "update packages from remote mirror",
		Long: `
Update downloads list of packages and package files.

ex:
  $ aptly mirror update wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-mirror-update", flag.ExitOnError),
	}

	cmd.Flag.Bool("ignore-checksums", false, "ignore checksum mismatches while downloading package files and metadata")
	cmd.Flag.Bool("ignore-signatures", false, "disable verification of Release file signatures")
	cmd.Flag.Var(&keyRings, "keyring", "gpg keyring to use when verifying Release file (could be specified multiple times)")

	return cmd
}
