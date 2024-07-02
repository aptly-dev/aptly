package cmd

import (
	"strings"

	"github.com/aptly-dev/aptly/pgp"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func getVerifier(flags *flag.FlagSet) (pgp.Verifier, error) {
	keyRings := flags.Lookup("keyring").Value.Get().([]string)
	ignoreSignatures := context.Config().GpgDisableVerify
	if context.Flags().IsSet("ignore-signatures") {
		ignoreSignatures = context.Flags().Lookup("ignore-signatures").Value.Get().(bool)
	}

	verifier := context.GetVerifier()
	for _, keyRing := range keyRings {
		verifier.AddKeyring(keyRing)
	}

	err := verifier.InitKeyring(ignoreSignatures == false) // be verbose only if verifying signatures is requested
	if err != nil {
		return nil, err
	}

	return verifier, nil
}

type keyRingsFlag struct {
	keyRings []string
}

func (k *keyRingsFlag) Set(value string) error {
	k.keyRings = append(k.keyRings, value)
	return nil
}

func (k *keyRingsFlag) Get() interface{} {
	return k.keyRings
}

func (k *keyRingsFlag) String() string {
	return strings.Join(k.keyRings, ",")
}

func makeCmdMirror() *commander.Command {
	return &commander.Command{
		UsageLine: "mirror",
		Short:     "manage mirrors of remote repositories",
		Subcommands: []*commander.Command{
			makeCmdMirrorCreate(),
			makeCmdMirrorList(),
			makeCmdMirrorShow(),
			makeCmdMirrorDrop(),
			makeCmdMirrorUpdate(),
			makeCmdMirrorRename(),
			makeCmdMirrorEdit(),
			makeCmdMirrorSearch(),
		},
	}
}
