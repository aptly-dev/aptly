package utils

import (
	"fmt"
)

// StringsIsSubset checks that subset is strict subset of full, and returns
// error formatted with errorFmt otherwise
func StringsIsSubset(subset, full []string, errorFmt string) error {
	for _, checked := range subset {
		found := false
		for _, s := range full {
			if checked == s {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf(errorFmt, checked)
		}
	}
	return nil
}

// StrSlicesEqual compares two slices for equality
func StrSlicesEqual(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	for i, s := range s1 {
		if s != s2[i] {
			return false
		}
	}

	return true
}

// StrMapsEqual compares two map[string]string
func StrMapsEqual(m1, m2 map[string]string) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k, v := range m1 {
		v2, ok := m2[k]
		if !ok || v != v2 {
			return false
		}
	}

	return true

}
