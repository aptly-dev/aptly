package cmd

import (
	"strings"

	"github.com/aptly-dev/aptly/pgp"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func getSigner(flags *flag.FlagSet) (pgp.Signer, error) {
	if LookupOption(context.Config().GpgDisableSign, flags, "skip-signing") {
		return nil, nil
	}

	signer := context.GetSigner()

	var gpgKeys []string

	// CLI args have priority over config
	cliKeys := flags.Lookup("gpg-key").Value.Get().([]string)
	if len(cliKeys) > 0 {
		gpgKeys = cliKeys
	} else if len(context.Config().GpgKeys) > 0 {
		gpgKeys = context.Config().GpgKeys
	}

	for _, gpgKey := range gpgKeys {
		signer.SetKey(gpgKey)
	}
	signer.SetKeyRing(flags.Lookup("keyring").Value.String(), flags.Lookup("secret-keyring").Value.String())
	signer.SetPassphrase(flags.Lookup("passphrase").Value.String(), flags.Lookup("passphrase-file").Value.String())
	signer.SetBatch(flags.Lookup("batch").Value.Get().(bool))

	err := signer.Init()
	if err != nil {
		return nil, err
	}

	return signer, nil

}

type gpgKeyFlag struct {
	gpgKeys []string
}

func (k *gpgKeyFlag) Set(value string) error {
	k.gpgKeys = append(k.gpgKeys, value)
	return nil
}

func (k *gpgKeyFlag) Get() interface{} {
	return k.gpgKeys
}

func (k *gpgKeyFlag) String() string {
	return strings.Join(k.gpgKeys, ",")
}

func makeCmdPublish() *commander.Command {
	return &commander.Command{
		UsageLine: "publish",
		Short:     "manage published repositories",
		Subcommands: []*commander.Command{
			makeCmdPublishDrop(),
			makeCmdPublishList(),
			makeCmdPublishRepo(),
			makeCmdPublishShow(),
			makeCmdPublishSnapshot(),
			makeCmdPublishSource(),
			makeCmdPublishSwitch(),
			makeCmdPublishUpdate(),
		},
	}
}

func makeCmdPublishSource() *commander.Command {
	return &commander.Command{
		UsageLine: "source",
		Short:     "manage sources of published repository",
		Subcommands: []*commander.Command{
			makeCmdPublishSourceAdd(),
			makeCmdPublishSourceDrop(),
			makeCmdPublishSourceList(),
			makeCmdPublishSourceRemove(),
			makeCmdPublishSourceReplace(),
			makeCmdPublishSourceUpdate(),
		},
	}
}
