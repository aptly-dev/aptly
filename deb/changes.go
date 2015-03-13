package deb

import (
	"fmt"
	"github.com/smira/aptly/utils"
	"os"
)

// Changes is a result of .changes file parsing
type Changes struct {
	Changes      string
	Distribution string
	Files        PackageFiles
}

// ParseChangesFile does optional signature verification and parses changes files
func ParseChangesFile(path string, acceptUnsigned, ignoreSignature bool, verifier utils.Verifier) (*Changes, error) {
	input, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer input.Close()

	isClearSigned, err := verifier.IsClearSigned(input)
	if err != nil {
		return nil, err
	}

	input.Seek(0, 0)

	if !isClearSigned && !acceptUnsigned {
		return nil, fmt.Errorf(".changes file is not signed and unsigned processing hasn't been enabled")
	}

	if isClearSigned && !ignoreSignature {
		err = verifier.VerifyClearsigned(input)
		if err != nil {
			return nil, err
		}
		input.Seek(0, 0)
	}

	var text *os.File

	if isClearSigned {
		text, err = verifier.ExtractClearsigned(input)
		if err != nil {
			return nil, err
		}
		defer text.Close()
	} else {
		text = input
	}

	reader := NewControlFileReader(text)
	stanza, err := reader.ReadStanza()
	if err != nil {
		return nil, err
	}

	result := &Changes{
		Distribution: stanza["Distribution"],
		Changes:      stanza["Changes"],
	}

	result.Files, err = result.Files.ParseSumFields(stanza)
	if err != nil {
		return nil, err
	}

	return result, nil
}
