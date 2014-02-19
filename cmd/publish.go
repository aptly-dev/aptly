package cmd

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/utils"
)

func getSigner(cmd *commander.Command) (utils.Signer, error) {
	if cmd.Flag.Lookup("skip-signing").Value.Get().(bool) || utils.Config.GpgDisableSign {
		return nil, nil
	}

	signer := &utils.GpgSigner{}
	signer.SetKey(cmd.Flag.Lookup("gpg-key").Value.String())
	signer.SetKeyRing(cmd.Flag.Lookup("keyring").Value.String(), cmd.Flag.Lookup("secret-keyring").Value.String())

	err := signer.Init()
	if err != nil {
		return nil, err
	}

	return signer, nil

}

func makeCmdPublish() *commander.Command {
	return &commander.Command{
		UsageLine: "publish",
		Short:     "manage published repositories",
		Subcommands: []*commander.Command{
			makeCmdPublishSnapshot(),
			makeCmdPublishList(),
			makeCmdPublishDrop(),
		},
		Flag: *flag.NewFlagSet("aptly-publish", flag.ExitOnError),
	}
}
