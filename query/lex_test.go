package query

import (
	"fmt"
	. "launchpad.net/gocheck"
)

type LexerSuite struct {
}

var _ = Suite(&LexerSuite{})

func (s *LexerSuite) TestLexing(c *C) {
	_, ch := lex("query", "package (<< 1.3), $Source | !app")

	c.Check(<-ch, Equals, item{typ: itemString, val: "package"})
	c.Check(<-ch, Equals, item{typ: itemLeftParen, val: "("})
	c.Check(<-ch, Equals, item{typ: itemLt, val: "<<"})
	c.Check(<-ch, Equals, item{typ: itemString, val: "1.3"})
	c.Check(<-ch, Equals, item{typ: itemRightParen, val: ")"})
	c.Check(<-ch, Equals, item{typ: itemAnd, val: ","})
	c.Check(<-ch, Equals, item{typ: itemString, val: "$Source"})
	c.Check(<-ch, Equals, item{typ: itemOr, val: "|"})
	c.Check(<-ch, Equals, item{typ: itemNot, val: "!"})
	c.Check(<-ch, Equals, item{typ: itemString, val: "app"})
	c.Check(<-ch, Equals, item{typ: itemEOF, val: ""})
}

func (s *LexerSuite) TestConsume(c *C) {
	l, _ := lex("query", "package (<< 1.3)")

	c.Check(l.Current(), Equals, item{typ: itemString, val: "package"})
	c.Check(l.Current(), Equals, item{typ: itemString, val: "package"})
	l.Consume()
	c.Check(l.Current(), Equals, item{typ: itemLeftParen, val: "("})
	l.Consume()
	c.Check(l.Current(), Equals, item{typ: itemLt, val: "<<"})
}

func (s *LexerSuite) TestString(c *C) {
	l, _ := lex("query", "package (<< 1.3)")

	c.Check(fmt.Sprintf("%s", l.Current()), Equals, "\"package\"")
	l.Consume()
	c.Check(fmt.Sprintf("%s", l.Current()), Equals, "(")
}
