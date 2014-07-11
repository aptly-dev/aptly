// Package query implements query language for
package query

import (
	"github.com/smira/aptly/deb"
)

/*

  Query language resembling Debian dependencies and reprepro
  queries: http://mirrorer.alioth.debian.org/reprepro.1.html

  Query := A | A '|' Query
  A := B | B ',' A
  B := C | '!' B
  C := '(' Query ')' | D
  D := <field> <condition> <arch_condition> | <pkg>_<version>_<arch>
  field := <package-name> | <field> | $special_field
  condition := '(' <operator> value ')' |
  arch_condition := '{' arch '}' |
  operator := | << | < | <= | > | >> | >= | = | % | ~
*/

// Parse parses input package query into PackageQuery tree ready for evaluation
func Parse(query string) (result deb.PackageQuery, err error) {
	l, _ := lex("", query)
	result, err = parse(l)
	return
}
