package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/smira/flag"
)

// StringOrFileFlag is a custom flag type that can handle both string input and file input.
// If the input starts with '@', it is treated as a filename and the contents are read from the file.
// If the input is '@-', the contents are read from stdin.
type StringOrFileFlag struct {
	value string
}

func (s *StringOrFileFlag) String() string {
	return s.value
}

func (s *StringOrFileFlag) Set(value string) error {
	var err error
	s.value, err = GetStringOrFileContent(value)
	return err
}

func (s *StringOrFileFlag) Get() any {
	return s.value
}

func AddStringOrFileFlag(flagSet *flag.FlagSet, name string, value string, usage string) *StringOrFileFlag {
	result := &StringOrFileFlag{value: value}
	flagSet.Var(result, name, usage)
	return result
}

func GetStringOrFileContent(value string) (string, error) {
	if !strings.HasPrefix(value, "@") {
		return value, nil
	}

	filename := strings.TrimPrefix(value, "@")
	var data []byte
	var err error
	if filename == "-" { // Read from stdin
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(filename)
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}
