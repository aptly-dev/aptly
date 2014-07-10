// Package query implements query language for
package query

/*

  Query language resembling Debian dependencies and reprepro
  queries: http://mirrorer.alioth.debian.org/reprepro.1.html

  Query := A | A '|' Query
  A := B | B ',' A
  B := C | '!' B
  C := '(' Query ')' | D
  D := <field> <condition>
  field := <package-name> | <field> | $special_field
  condition := '(' <operator> value ')' |
  operator := | << | < | <= | > | >> | >= | = | % | ~
*/
