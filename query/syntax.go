package query

import (
	"fmt"
	"github.com/smira/aptly/deb"
	"strings"
	"unicode"
	"unicode/utf8"
)

type parser struct {
	name  string // used only for error reports.
	input *lexer // the input lexer
	err   error  // error stored while parsing
}

func parse(input *lexer) (PackageQuery, error) {
	p := &parser{
		name:  input.name,
		input: input,
	}
	query := p.parse()
	if p.err != nil {
		return nil, p.err
	}
	return query, nil
}

// Entry into parser
func (p *parser) parse() PackageQuery {
	defer func() {
		if r := recover(); r != nil {
			p.err = fmt.Errorf("parsing failed: %s", r)
		}
	}()

	q := p.Query()
	if p.input.Current().typ != itemEOF {
		panic(fmt.Sprintf("unexpected token %s: expecting end of query", p.input.Current()))
	}
	return q
}

// Query := A | A '|' Query
func (p *parser) Query() PackageQuery {
	q := p.A()
	if p.input.Current().typ == itemOr {
		p.input.Consume()
		return &OrQuery{L: q, R: p.Query()}
	}
	return q
}

// A := B | B ',' A
func (p *parser) A() PackageQuery {
	q := p.B()
	if p.input.Current().typ == itemAnd {
		p.input.Consume()
		return &AndQuery{L: q, R: p.A()}
	}
	return q
}

// B := C | '!' B
func (p *parser) B() PackageQuery {
	if p.input.Current().typ == itemNot {
		p.input.Consume()
		return &NotQuery{Q: p.B()}
	}
	return p.C()
}

// C := '(' Query ')' | D
func (p *parser) C() PackageQuery {
	if p.input.Current().typ == itemLeftParen {
		p.input.Consume()
		q := p.Query()
		if p.input.Current().typ != itemRightParen {
			panic(fmt.Sprintf("unexpected token %s: expecting ')'", p.input.Current()))
		}
		p.input.Consume()
		return q
	}
	return p.D()
}

func operatorToRelation(operator itemType) int {
	switch operator {
	case 0:
		return deb.VersionDontCare
	case itemLt:
		return deb.VersionLess
	case itemLtEq:
		return deb.VersionLessOrEqual
	case itemGt:
		return deb.VersionGreater
	case itemGtEq:
		return deb.VersionGreaterOrEqual
	case itemEq:
		return deb.VersionEqual
	case itemPatMatch:
		return deb.VersionPatternMatch
	case itemRegexp:
		return deb.VersionRegexp
	}
	panic("unable to map token to relation")
}

// D := <field> <condition>
// field := <package-name> | <field> | $special_field
func (p *parser) D() PackageQuery {
	if p.input.Current().typ != itemString {
		panic(fmt.Sprintf("unexpected token %s: expecting field or package name", p.input.Current()))
	}

	field := p.input.Current().val
	p.input.Consume()

	operator, value := p.Condition()

	r, _ := utf8.DecodeRuneInString(field)
	if strings.HasPrefix(field, "$") || unicode.IsUpper(r) {
		// special field or regular field
		return &FieldQuery{Field: field, Relation: operatorToRelation(operator), Value: value}
	}

	// regular dependency-like query
	return &DependencyQuery{Dep: deb.Dependency{Pkg: field, Relation: operatorToRelation(operator), Version: value}}
}

// condition := '(' <operator> value ')' |
// operator := | << | < | <= | > | >> | >= | = | % | ~
func (p *parser) Condition() (operator itemType, value string) {
	if p.input.Current().typ != itemLeftParen {
		return
	}
	p.input.Consume()

	if p.input.Current().typ == itemLt ||
		p.input.Current().typ == itemGt ||
		p.input.Current().typ == itemLtEq ||
		p.input.Current().typ == itemGtEq ||
		p.input.Current().typ == itemEq ||
		p.input.Current().typ == itemPatMatch ||
		p.input.Current().typ == itemRegexp {
		operator = p.input.Current().typ
		p.input.Consume()
	} else {
		operator = itemEq
	}

	if p.input.Current().typ != itemString {
		panic(fmt.Sprintf("unexpected token %s: expecting value", p.input.Current()))
	}
	value = p.input.Current().val
	p.input.Consume()

	if p.input.Current().typ != itemRightParen {
		panic(fmt.Sprintf("unexpected token %s: expecting ')'", p.input.Current()))
	}
	p.input.Consume()

	return
}
