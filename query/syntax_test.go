package query

import (
	"github.com/smira/aptly/deb"
	. "launchpad.net/gocheck"
)

type SyntaxSuite struct {
}

var _ = Suite(&SyntaxSuite{})

func (s *SyntaxSuite) TestParsing(c *C) {
	l, _ := lex("query", "package (<< 1.3), $Source")
	q, err := parse(l)

	c.Assert(err, IsNil)
	c.Check(q.(*deb.AndQuery).L, DeepEquals, &deb.DependencyQuery{Dep: deb.Dependency{Pkg: "package", Relation: deb.VersionLess, Version: "1.3"}})
	c.Check(q.(*deb.AndQuery).R, DeepEquals, &deb.FieldQuery{Field: "$Source"})

	l, _ = lex("query", "package (1.3), Name (lala) | !$Source")
	q, err = parse(l)

	c.Assert(err, IsNil)
	c.Check(q.(*deb.OrQuery).L.(*deb.AndQuery).L, DeepEquals, &deb.DependencyQuery{Dep: deb.Dependency{Pkg: "package", Relation: deb.VersionEqual, Version: "1.3"}})
	c.Check(q.(*deb.OrQuery).L.(*deb.AndQuery).R, DeepEquals, &deb.FieldQuery{Field: "Name", Relation: deb.VersionEqual, Value: "lala"})
	c.Check(q.(*deb.OrQuery).R.(*deb.NotQuery).Q, DeepEquals, &deb.FieldQuery{Field: "$Source"})

	l, _ = lex("query", "package, ((!(Name | $Source (~ a.*))))")
	q, err = parse(l)

	c.Assert(err, IsNil)
	c.Check(q.(*deb.AndQuery).L, DeepEquals, &deb.DependencyQuery{Dep: deb.Dependency{Pkg: "package", Relation: deb.VersionDontCare}})
	c.Check(q.(*deb.AndQuery).R.(*deb.NotQuery).Q.(*deb.OrQuery).L, DeepEquals, &deb.FieldQuery{Field: "Name", Relation: deb.VersionDontCare})
	c.Check(q.(*deb.AndQuery).R.(*deb.NotQuery).Q.(*deb.OrQuery).R, DeepEquals, &deb.FieldQuery{Field: "$Source", Relation: deb.VersionRegexp, Value: "a.*"})

	l, _ = lex("query", "package (> 5.3.7)")
	q, err = parse(l)

	c.Assert(err, IsNil)
	c.Check(q, DeepEquals, &deb.DependencyQuery{Dep: deb.Dependency{Pkg: "package", Relation: deb.VersionGreaterOrEqual, Version: "5.3.7"}})
}

func (s *SyntaxSuite) TestParsingErrors(c *C) {
	l, _ := lex("query", "package (> 5.3.7), ")
	_, err := parse(l)
	c.Check(err, ErrorMatches, "parsing failed: unexpected token <EOL>: expecting field or package name")

	l, _ = lex("query", "package>5.3.7)")
	_, err = parse(l)
	c.Check(err, ErrorMatches, "parsing failed: unexpected token >: expecting end of query")

	l, _ = lex("query", "package | !|")
	_, err = parse(l)
	c.Check(err, ErrorMatches, "parsing failed: unexpected token |: expecting field or package name")

	l, _ = lex("query", "((package )")
	_, err = parse(l)
	c.Check(err, ErrorMatches, "parsing failed: unexpected token <EOL>: expecting '\\)'")

	l, _ = lex("query", "!package )")
	_, err = parse(l)
	c.Check(err, ErrorMatches, "parsing failed: unexpected token \\): expecting end of query")
}
