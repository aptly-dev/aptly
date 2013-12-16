package debian

import (
	"bytes"
	debc "github.com/smira/godebiancontrol"
	"github.com/ugorji/go/codec"
	"strings"
)

// Package is single instance of Debian package
//
// TODO: support source & binary
type Package struct {
	Name         string
	Version      string
	Filename     string
	Architecture string
	Depends      []string
	PreDepends   []string
	Suggests     []string
	Recommends   []string
	Extra        debc.Paragraph
}

func parseDependencies(input debc.Paragraph, key string) []string {
	value, ok := input[key]
	if !ok {
		return nil
	}

	delete(input, key)

	return strings.Split(value, ", ")
}

// NewPackageFromControlFile creates Package from parsed Debian control file
func NewPackageFromControlFile(input debc.Paragraph) *Package {
	result := &Package{
		Name:         input["Package"],
		Version:      input["Version"],
		Filename:     input["Filename"],
		Architecture: input["Architecture"],
	}

	delete(input, "Package")
	delete(input, "Version")
	delete(input, "Filename")
	delete(input, "Architecture")

	result.Depends = parseDependencies(input, "Depends")
	result.PreDepends = parseDependencies(input, "Pre-Depends")
	result.Suggests = parseDependencies(input, "Suggests")
	result.Recommends = parseDependencies(input, "Recommends")

	result.Extra = input

	return result
}

// Key returns unique key identifying package
func (p *Package) Key() []byte {
	return []byte(p.Name + " " + p.Version)
}

// Encode does msgpack encoding of Package
func (p *Package) Encode() []byte {
	var buf bytes.Buffer

	encoder := codec.NewEncoder(&buf, &codec.MsgpackHandle{})
	encoder.Encode(p)

	return buf.Bytes()
}

// Decode decodes msgpack representation into Package
func (p *Package) Decode(input []byte) error {
	decoder := codec.NewDecoderBytes(input, &codec.MsgpackHandle{})
	return decoder.Decode(p)
}
