package deb

import (
	"strings"
)

// PackageDependencies are various parsed dependencies
type PackageDependencies struct {
	Depends           []string
	BuildDepends      []string
	BuildDependsInDep []string
	PreDepends        []string
	Suggests          []string
	Recommends        []string
}

func parseDependencies(input Stanza, key string) []string {
	value, ok := input[key]
	if !ok {
		return nil
	}

	delete(input, key)

	result := strings.Split(value, ",")
	for i := range result {
		result[i] = strings.TrimSpace(result[i])
	}
	return result
}
