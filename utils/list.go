package utils

import (
	"fmt"
)

// StringsIsSubset checks that subset is strict subset of full, and returns
// error formatted with errorFmt otherwise
func StringsIsSubset(subset []string, full []string, errorFmt string) error {
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
