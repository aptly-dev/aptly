package deb

import (
	"github.com/smira/aptly/utils"
)

// UploadersRule is single rule of format: what packages can group or key upload
type UploadersRule struct {
	Condition         string       `json:"condition"`
	Allow             []string     `json:"allow"`
	Deny              []string     `json:"deny"`
	CompiledCondition PackageQuery `json:"-"`
}

// Uploaders is configuration of restrictions for .changes file importing
type Uploaders struct {
	Groups map[string][]string `json:"groups"`
	Rules  []UploadersRule     `json:"rules"`
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
func (u *Uploaders) IsAllowed(changes *Changes) bool {
	for _, rule := range u.Rules {
		if rule.CompiledCondition.Matches(changes) {
			deny := u.ExpandGroups(rule.Deny)
			for _, key := range changes.SignatureKeys {
				for _, item := range deny {
					if item == "*" || key.Matches(utils.GpgKey(item)) {
						return false
					}
				}
			}

			allow := u.ExpandGroups(rule.Allow)
			for _, key := range changes.SignatureKeys {
				for _, item := range allow {
					if item == "*" || key.Matches(utils.GpgKey(item)) {
						return true
					}
				}
			}
		}
	}

	return false
}
