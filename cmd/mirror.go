package cmd

import (
	"github.com/smira/aptly/utils"
	"github.com/smira/commander"
	"github.com/smira/flag"
	"strings"
)

func getVerifier(flags *flag.FlagSet) (utils.Verifier, error) {
	if context.Config().GpgDisableVerify || flags.Lookup("ignore-signatures").Value.Get().(bool) {
		return nil, nil
	}

	keyRings := flags.Lookup("keyring").Value.Get().([]string)

	verifier := &utils.GpgVerifier{}
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
		},
	}
}
