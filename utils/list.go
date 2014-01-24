package utils

import (
	"fmt"
	"sort"
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

// StrSliceHasItem checks item for presence in slice
func StrSliceHasItem(s []string, item string) bool {
	for _, v := range s {
		if v == item {
			return true
		}
	}
	return false
}

// StrMapSortedKeys returns keys of map[string]string sorted
func StrMapSortedKeys(m map[string]string) []string {
	keys := make(sort.StringSlice, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

// StrSliceDeduplicate removes dups in slice
func StrSliceDeduplicate(s []string) []string {
	l := len(s)
	if l < 2 {
		return s
	}
	if l == 2 {
		if s[0] == s[1] {
			return s[0:1]
		}
		return s
	}

	found := make(map[string]bool, l)
	j := 0
	for i, x := range s {
		if !found[x] {
			found[x] = true
			s[j] = s[i]
			j++
		}
	}

	return s[:j]
}
