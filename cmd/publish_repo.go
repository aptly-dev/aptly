package cmd

import (
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func makeCmdPublishRepo() *commander.Command {
	cmd := &commander.Command{
		Run:       aptlyPublishSnapshotOrRepo,
		UsageLine: "repo <name> [[<endpoint>:]<prefix>]",
		Short:     "publish local repository",
		Long: `
Command publishes current state of local repository ready to be consumed
by apt tools. Published repostiories appear under rootDir/public directory.
Valid GPG key is required for publishing.

Multiple component repository could be published by specifying several
components split by commas via -component flag and multiple local
repositories as the arguments:

    aptly publish repo -component=main,contrib repo-main repo-contrib

It is not recommended to publish local repositories directly unless the
repository is for testing purposes and changes happen frequently. For
production usage please take snapshot of repository and publish it
using publish snapshot command.

Example:

    $ aptly publish repo testing
`,
		Flag: *flag.NewFlagSet("aptly-publish-repo", flag.ExitOnError),
	}
	cmd.Flag.String("distribution", "", "distribution name to publish")
	cmd.Flag.String("component", "", "component name to publish (for multi-component publishing, separate components with commas)")
	cmd.Flag.String("gpg-key", "", "GPG key ID to use when signing the release")
	cmd.Flag.Var(&keyRingsFlag{}, "keyring", "GPG keyring to use (instead of default)")
	cmd.Flag.String("secret-keyring", "", "GPG secret keyring to use (instead of default)")
	cmd.Flag.String("passphrase", "", "GPG passphrase for the key (warning: could be insecure)")
	cmd.Flag.String("passphrase-file", "", "GPG passphrase-file for the key (warning: could be insecure)")
	cmd.Flag.Bool("batch", false, "run GPG with detached tty")
	cmd.Flag.Bool("skip-signing", false, "don't sign Release files with GPG")
	cmd.Flag.Bool("skip-contents", false, "don't generate Contents indexes")
	cmd.Flag.Bool("skip-bz2", false, "don't generate bzipped indexes")
	cmd.Flag.String("origin", "", "origin name to publish")
	cmd.Flag.String("notautomatic", "", "set value for NotAutomatic field")
	cmd.Flag.String("butautomaticupgrades", "", "set  value for ButAutomaticUpgrades field")
	cmd.Flag.String("label", "", "label to publish")
	cmd.Flag.String("suite", "", "suite to publish (defaults to distribution)")
	cmd.Flag.String("codename", "", "codename to publish (defaults to distribution)")
	cmd.Flag.Bool("force-overwrite", false, "overwrite files in package pool in case of mismatch")
	cmd.Flag.Bool("acquire-by-hash", false, "provide index files by hash")

	return cmd
}
