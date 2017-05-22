package deb

import (
	"github.com/smira/aptly/pgp"
	. "gopkg.in/check.v1"
)

type UploadersSuite struct {
}

var _ = Suite(&UploadersSuite{})

func (s *UploadersSuite) TestExpandGroups(c *C) {
	u := &Uploaders{
		Groups: map[string][]string{
			"group1": {"key1", "group2"},
			"group2": {"key1", "key2", "key3", "group3"},
			"group3": {},
			"group4": {"key1", "group5"},
			"group6": {"key1", "group8"},
			"group7": {"key2", "group6"},
			"group8": {"group7"},
		},
	}

	c.Check(u.ExpandGroups([]string{"group1"}), DeepEquals, []string{"key1", "key2", "key3"})
	c.Check(u.ExpandGroups([]string{"group2"}), DeepEquals, []string{"key1", "key2", "key3"})
	c.Check(u.ExpandGroups([]string{"group3"}), DeepEquals, []string{})
	c.Check(u.ExpandGroups([]string{"group4"}), DeepEquals, []string{"key1", "group5"})
	c.Check(u.ExpandGroups([]string{"group6"}), DeepEquals, []string{"key1", "key2"})
	c.Check(u.ExpandGroups([]string{"group7"}), DeepEquals, []string{"key2", "key1"})
	c.Check(u.ExpandGroups([]string{"group8"}), DeepEquals, []string{"key2", "key1"})
}

func (s *UploadersSuite) TestIsAllowed(c *C) {
	u := &Uploaders{
		Groups: map[string][]string{
			"group1": {"37E1C17570096AD1", "EC4B033C70096AD1"},
		},
		Rules: []UploadersRule{
			{
				CompiledCondition: &FieldQuery{Field: "Source", Relation: VersionEqual, Value: "calamares"},
				Allow:             []string{"*"},
			},
			{
				CompiledCondition: &FieldQuery{Field: "Source", Relation: VersionEqual, Value: "never-calamares"},
				Deny:              []string{"*"},
			},
			{
				CompiledCondition: &FieldQuery{Field: "Source", Relation: VersionEqual, Value: "some-calamares"},
				Allow:             []string{"group1", "12345678"},
			},
			{
				CompiledCondition: &FieldQuery{Field: "Source", Relation: VersionEqual, Value: "some-calamares"},
				Deny:              []string{"45678901", "12345678"},
			},
		},
	}

	// no keys - not allowed
	c.Check(u.IsAllowed(&Changes{SignatureKeys: []pgp.Key{}, Stanza: Stanza{"Source": "calamares"}}), ErrorMatches, "denied as no rule matches")

	// no rule - not allowed
	c.Check(u.IsAllowed(&Changes{SignatureKeys: []pgp.Key{"37E1C17570096AD1", "EC4B033C70096AD1"}, Stanza: Stanza{"Source": "unknown-calamares"}}), ErrorMatches, "denied as no rule matches")

	// first rule: allow anyone do stuff with calamares
	c.Check(u.IsAllowed(&Changes{SignatureKeys: []pgp.Key{"ABCD1234", "1234ABCD"}, Stanza: Stanza{"Source": "calamares"}}), IsNil)

	// second rule: nobody is allowed to do stuff with never-calamares
	c.Check(u.IsAllowed(&Changes{SignatureKeys: []pgp.Key{"ABCD1234", "1234ABCD"}, Stanza: Stanza{"Source": "never-calamares"}}),
		ErrorMatches, "denied according to rule: {\"condition\":\"\",\"allow\":null,\"deny\":\\[\"\\*\"\\]}")

	// third rule: anyone from the group or explicit key
	c.Check(u.IsAllowed(&Changes{SignatureKeys: []pgp.Key{"45678901", "12345678"}, Stanza: Stanza{"Source": "some-calamares"}}), IsNil)
	c.Check(u.IsAllowed(&Changes{SignatureKeys: []pgp.Key{"37E1C17570096AD1"}, Stanza: Stanza{"Source": "some-calamares"}}), IsNil)
	c.Check(u.IsAllowed(&Changes{SignatureKeys: []pgp.Key{"70096AD1"}, Stanza: Stanza{"Source": "some-calamares"}}), IsNil)

	// fourth rule: some are not allowed
	c.Check(u.IsAllowed(&Changes{SignatureKeys: []pgp.Key{"ABCD1234", "45678901"}, Stanza: Stanza{"Source": "some-calamares"}}),
		ErrorMatches, "denied according to rule: {\"condition\":\"\",\"allow\":null,\"deny\":\\[\"45678901\",\"12345678\"\\]}")
}
