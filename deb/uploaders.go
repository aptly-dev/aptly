package deb

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/DisposaBoy/JsonConfigReader"
	"github.com/smira/aptly/pgp"
	"github.com/smira/aptly/utils"
)

// UploadersRule is single rule of format: what packages can group or key upload
type UploadersRule struct {
	Condition         string       `json:"condition"`
	Allow             []string     `json:"allow"`
	Deny              []string     `json:"deny"`
	CompiledCondition PackageQuery `json:"-" codec:"-"`
}

func (u UploadersRule) String() string {
	b, _ := json.Marshal(u)
	return string(b)
}

// Uploaders is configuration of restrictions for .changes file importing
type Uploaders struct {
	Groups map[string][]string `json:"groups"`
	Rules  []UploadersRule     `json:"rules"`
}

func (u *Uploaders) String() string {
	b, _ := json.Marshal(u)
	return string(b)
}

// NewUploadersFromFile loads Uploaders structue from .json file
func NewUploadersFromFile(path string) (*Uploaders, error) {
	uploaders := &Uploaders{}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error loading uploaders file: %s", err)
	}
	defer f.Close()

	err = json.NewDecoder(JsonConfigReader.New(f)).Decode(&uploaders)
	if err != nil {
		return nil, fmt.Errorf("error loading uploaders file: %s", err)
	}

	return uploaders, nil
}

func (u *Uploaders) expandGroupsInternal(items []string, trail []string) []string {
	result := []string{}

	for _, item := range items {
		// stop infinite recursion
		if utils.StrSliceHasItem(trail, item) {
			continue
		}

		group, ok := u.Groups[item]
		if !ok {
			result = append(result, item)
		} else {
			newTrail := append([]string(nil), trail...)
			result = append(result, u.expandGroupsInternal(group, append(newTrail, item))...)
		}
	}

	return result
}

// ExpandGroups expands list of keys/groups into list of keys
func (u *Uploaders) ExpandGroups(items []string) []string {
	result := u.expandGroupsInternal(items, []string{})

	return utils.StrSliceDeduplicate(result)
}

// IsAllowed checks whether listed keys are allowed to upload given .changes file
func (u *Uploaders) IsAllowed(changes *Changes) error {
	for _, rule := range u.Rules {
		if rule.CompiledCondition.Matches(changes) {
			deny := u.ExpandGroups(rule.Deny)
			for _, key := range changes.SignatureKeys {
				for _, item := range deny {
					if item == "*" || key.Matches(pgp.Key(item)) {
						return fmt.Errorf("denied according to rule: %s", rule)
					}
				}
			}

			allow := u.ExpandGroups(rule.Allow)
			for _, key := range changes.SignatureKeys {
				for _, item := range allow {
					if item == "*" || key.Matches(pgp.Key(item)) {
						return nil
					}
				}
			}
		}
	}

	return fmt.Errorf("denied as no rule matches")
}
