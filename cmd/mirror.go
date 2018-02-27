package cmd

import (
	"strings"

	"github.com/smira/aptly/pgp"
	"github.com/smira/commander"
	"github.com/smira/flag"
)

func getVerifier(flags *flag.FlagSet) (pgp.Verifier, error) {
	if LookupOption(context.Config().GpgDisableVerify, flags, "ignore-signatures") {
		return nil, nil
	}

	keyRings := flags.Lookup("keyring").Value.Get().([]string)

	verifier := context.GetVerifier()
	for _, keyRing := range keyRings {
		verifier.AddKeyring(keyRing)
	}

	err := verifier.InitKeyring()
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
