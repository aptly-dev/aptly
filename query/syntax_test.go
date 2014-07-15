package query

import (
	"github.com/smira/aptly/deb"
	. "launchpad.net/gocheck"
	"regexp"
)

type SyntaxSuite struct {
}

var _ = Suite(&SyntaxSuite{})

func (s *SyntaxSuite) TestParsing(c *C) {
	l, _ := lex("query", "package (<< 1.3~dev), $Source")
	q, err := parse(l)

	c.Assert(err, IsNil)
	c.Check(q.(*deb.AndQuery).L, DeepEquals, &deb.DependencyQuery{Dep: deb.Dependency{Pkg: "package", Relation: deb.VersionLess, Version: "1.3~dev"}})
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
	c.Check(q.(*deb.AndQuery).R.(*deb.NotQuery).Q.(*deb.OrQuery).R, DeepEquals, &deb.FieldQuery{Field: "$Source", Relation: deb.VersionRegexp, Value: "a.*",
		Regexp: regexp.MustCompile("a.*")})

	l, _ = lex("query", "package (> 5.3.7)")
	q, err = parse(l)

	c.Assert(err, IsNil)
	c.Check(q, DeepEquals, &deb.DependencyQuery{Dep: deb.Dependency{Pkg: "package", Relation: deb.VersionGreaterOrEqual, Version: "5.3.7"}})

	l, _ = lex("query", "package (~ 5\\.3.*~dev)")
	q, err = parse(l)

	c.Assert(err, IsNil)
	c.Check(q, DeepEquals, &deb.DependencyQuery{Dep: deb.Dependency{Pkg: "package", Relation: deb.VersionRegexp, Version: "5\\.3.*~dev",
		Regexp: regexp.MustCompile("5\\.3.*~dev")}})

	l, _ = lex("query", "alien-data_1.3.4~dev_i386")
	q, err = parse(l)

	c.Assert(err, IsNil)
	c.Check(q, DeepEquals, &deb.PkgQuery{Pkg: "alien-data", Version: "1.3.4~dev", Arch: "i386"})

	l, _ = lex("query", "package (> 5.3.7) {amd64}")
	q, err = parse(l)

	c.Assert(err, IsNil)
	c.Check(q, DeepEquals, &deb.DependencyQuery{
		Dep: deb.Dependency{Pkg: "package", Relation: deb.VersionGreaterOrEqual, Version: "5.3.7", Architecture: "amd64"}})
}

func (s *SyntaxSuite) TestParsingErrors(c *C) {
	l, _ := lex("query", "package (> 5.3.7), ")
	_, err := parse(l)
	c.Check(err, ErrorMatches, "parsing failed: unexpected token <EOL>: expecting field or package name")

	l, _ = lex("query", "package>5.3.7)")
	_, err = parse(l)
	c.Check(err, ErrorMatches, "parsing failed: unexpected token \\): expecting end of query")

	l, _ = lex("query", "package | !|")
	_, err = parse(l)
	c.Check(err, ErrorMatches, "parsing failed: unexpected token |: expecting field or package name")

	l, _ = lex("query", "((package )")
	_, err = parse(l)
	c.Check(err, ErrorMatches, "parsing failed: unexpected token <EOL>: expecting '\\)'")

	l, _ = lex("query", "!package )")
	_, err = parse(l)
	c.Check(err, ErrorMatches, "parsing failed: unexpected token \\): expecting end of query")

	l, _ = lex("query", "'package )")
	_, err = parse(l)
	c.Check(err, ErrorMatches, "parsing failed: unexpected token error: unexpected eof in quoted string: expecting field or package name")

	l, _ = lex("query", "package (~ 1.2[34)")
	_, err = parse(l)
	c.Check(err, ErrorMatches, "parsing failed: regexp compile failed: error parsing regexp: missing closing \\]: `\\[34`")

	l, _ = lex("query", "$Name (~ 1.2[34)")
	_, err = parse(l)
	c.Check(err, ErrorMatches, "parsing failed: regexp compile failed: error parsing regexp: missing closing \\]: `\\[34`")
}
