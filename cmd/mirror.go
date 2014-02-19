package cmd

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/smira/aptly/utils"
	"strings"
)

func getVerifier(cmd *commander.Command) (utils.Verifier, error) {
	if utils.Config.GpgDisableVerify || cmd.Flag.Lookup("ignore-signatures").Value.Get().(bool) {
		return nil, nil
	}

	verifier := &utils.GpgVerifier{}
	for _, keyRing := range keyRings.keyRings {
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

var keyRings = keyRingsFlag{}

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
		},
		Flag: *flag.NewFlagSet("aptly-mirror", flag.ExitOnError),
	}
}
