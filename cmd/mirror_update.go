package cmd

import (
	"fmt"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func aptlyMirrorUpdates(cmd *commander.Command, args []string) error {
	var err error
	if len(args) != 1 {
		cmd.Usage()
		return err
	}
  repo_regexp := args[0]
  repo, err := context.CollectionFactory().RemoteRepoCollection().ByRegexp(repo_regexp)
  if nil != repo{
  for e := repo.Front(); e != nil; e = e.Next() {
      // there is probably a better way to do directly use e.Value as a string...  
      result := fmt.Sprintf("%s", e.Value)
      aptlyMirrorUpdate(cmd, []string{result})
    }
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

	repo, err := context.CollectionFactory().RemoteRepoCollection().ByName(name)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = context.CollectionFactory().RemoteRepoCollection().LoadComplete(repo)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	ignoreMismatch := context.flags.Lookup("ignore-checksums").Value.Get().(bool)

	verifier, err := getVerifier(context.flags)
	if err != nil {
		return fmt.Errorf("unable to initialize GPG verifier: %s", err)
	}

	err = repo.Fetch(context.Downloader(), verifier)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = repo.Download(context.Progress(), context.Downloader(), context.CollectionFactory(), context.PackagePool(), ignoreMismatch)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	err = context.CollectionFactory().RemoteRepoCollection().Update(repo)
	if err != nil {
		return fmt.Errorf("unable to update: %s", err)
	}

	context.Progress().Printf("\nMirror `%s` has been successfully updated.\n", repo.Name)
	return err
}

func makeCmdMirrorUpdate() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyMirrorUpdates,
		UsageLine: "update <name>",
		Short:     "update mirror",
		Long: `
Updates remote mirror (downloads package files and meta information). When mirror is created,
this command should be run for the first time to fetch mirror contents. This command could be
run many times to get updated repository contents. If interrupted, command could be restarted safely.

Example:

  $ aptly mirror update wheezy-main
`,
		Flag: *flag.NewFlagSet("aptly-mirror-update", flag.ExitOnError),
	}

	cmd.Flag.Bool("ignore-checksums", false, "ignore checksum mismatches while downloading package files and metadata")
	cmd.Flag.Bool("ignore-signatures", false, "disable verification of Release file signatures")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "gpg keyring to use when verifying Release file (could be specified multiple times)")

	return cmd
}
