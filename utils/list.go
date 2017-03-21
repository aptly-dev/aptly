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
	keys := make([]string, len(m))
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

// StrSlicesSubstract finds all the strings which are in l but not in r, both slices shoult be sorted
func StrSlicesSubstract(l, r []string) []string {
	var result []string

	// pointer to left and right reflists
	il, ir := 0, 0
	// length of reflists
	ll, lr := len(l), len(r)

	for il < ll || ir < lr {
		if il == ll {
			// left list exhausted, we got the result
			break
		}
		if ir == lr {
			// right list exhausted, append what is left to result
			result = append(result, l[il:]...)
			break
		}

		if l[il] == r[ir] {
			// r contains entry from l, so we skip it
			il++
			ir++
		} else if l[il] < r[ir] {
			// item il is not in r, append
			result = append(result, l[il])
			il++
		} else {
			// skip over to next item in r
			ir++
		}
	}

	return result
}
