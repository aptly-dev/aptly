package cmd

import (
	"github.com/smira/aptly/pgp"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func getSigner(flags *flag.FlagSet) (pgp.Signer, error) {
	if LookupOption(context.Config().GpgDisableSign, flags, "skip-signing") {
		return nil, nil
	}

	signer := context.GetSigner()
	signer.SetKey(flags.Lookup("gpg-key").Value.String())
	signer.SetKeyRing(flags.Lookup("keyring").Value.String(), flags.Lookup("secret-keyring").Value.String())
	signer.SetPassphrase(flags.Lookup("passphrase").Value.String(), flags.Lookup("passphrase-file").Value.String())
	signer.SetBatch(flags.Lookup("batch").Value.Get().(bool))

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
			makeCmdPublishDrop(),
			makeCmdPublishList(),
			makeCmdPublishRepo(),
			makeCmdPublishSnapshot(),
			makeCmdPublishSwitch(),
			makeCmdPublishUpdate(),
			makeCmdPublishShow(),
		},
	}
}
