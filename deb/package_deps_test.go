package deb

import (
	. "gopkg.in/check.v1"
)

type PackageDependenciesSuite struct{}

var _ = Suite(&PackageDependenciesSuite{})

func (s *PackageDependenciesSuite) TestParseDependenciesBasic(c *C) {
	// Test basic dependency parsing with single dependency
	stanza := Stanza{
		"Depends": "package1",
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, DeepEquals, []string{"package1"})

	// Check that key was removed from stanza
	_, exists := stanza["Depends"]
	c.Check(exists, Equals, false)
}

func (s *PackageDependenciesSuite) TestParseDependenciesMultiple(c *C) {
	// Test parsing multiple dependencies separated by commas
	stanza := Stanza{
		"Depends": "package1, package2, package3",
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, DeepEquals, []string{"package1", "package2", "package3"})
}

func (s *PackageDependenciesSuite) TestParseDependenciesWithVersions(c *C) {
	// Test parsing dependencies with version constraints
	stanza := Stanza{
		"Depends": "package1 (>= 1.0), package2 (<< 2.0), package3 (= 1.5)",
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, DeepEquals, []string{"package1 (>= 1.0)", "package2 (<< 2.0)", "package3 (= 1.5)"})
}

func (s *PackageDependenciesSuite) TestParseDependenciesWithWhitespace(c *C) {
	// Test parsing dependencies with various whitespace patterns
	stanza := Stanza{
		"Depends": "  package1  ,   package2   ,package3,  package4  ",
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, DeepEquals, []string{"package1", "package2", "package3", "package4"})
}

func (s *PackageDependenciesSuite) TestParseDependenciesEmpty(c *C) {
	// Test parsing empty dependency string
	stanza := Stanza{
		"Depends": "",
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, IsNil)

	// Check that key was removed from stanza
	_, exists := stanza["Depends"]
	c.Check(exists, Equals, false)
}

func (s *PackageDependenciesSuite) TestParseDependenciesWhitespaceOnly(c *C) {
	// Test parsing dependency string with only whitespace
	stanza := Stanza{
		"Depends": "   \t  \n  ",
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, IsNil)
}

func (s *PackageDependenciesSuite) TestParseDependenciesMissingKey(c *C) {
	// Test parsing when key doesn't exist in stanza
	stanza := Stanza{
		"SomeOtherField": "value",
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, IsNil)

	// Check that original stanza is unchanged
	_, exists := stanza["SomeOtherField"]
	c.Check(exists, Equals, true)
}

func (s *PackageDependenciesSuite) TestParseDependenciesComplexFormat(c *C) {
	// Test parsing complex dependency formats
	stanza := Stanza{
		"Depends": "libc6 (>= 2.17), libgcc1 (>= 1:4.1.1), libstdc++6 (>= 4.8.1)",
	}

	result := parseDependencies(stanza, "Depends")
	expected := []string{
		"libc6 (>= 2.17)",
		"libgcc1 (>= 1:4.1.1)",
		"libstdc++6 (>= 4.8.1)",
	}
	c.Check(result, DeepEquals, expected)
}

func (s *PackageDependenciesSuite) TestParseDependenciesAlternatives(c *C) {
	// Test parsing dependencies with alternatives (| separator within single dependency)
	stanza := Stanza{
		"Depends": "mail-transport-agent | postfix, libc6 (>= 2.17)",
	}

	result := parseDependencies(stanza, "Depends")
	expected := []string{
		"mail-transport-agent | postfix",
		"libc6 (>= 2.17)",
	}
	c.Check(result, DeepEquals, expected)
}

func (s *PackageDependenciesSuite) TestParseDependenciesSpecialCharacters(c *C) {
	// Test parsing dependencies with special characters in package names
	stanza := Stanza{
		"Depends": "lib-package++-dev, package.name, package_underscore",
	}

	result := parseDependencies(stanza, "Depends")
	expected := []string{
		"lib-package++-dev",
		"package.name",
		"package_underscore",
	}
	c.Check(result, DeepEquals, expected)
}

func (s *PackageDependenciesSuite) TestParseDependenciesArchitectures(c *C) {
	// Test parsing dependencies with architecture specifications
	stanza := Stanza{
		"Depends": "package1 [amd64], package2 [!arm64], package3 [i386 amd64]",
	}

	result := parseDependencies(stanza, "Depends")
	expected := []string{
		"package1 [amd64]",
		"package2 [!arm64]",
		"package3 [i386 amd64]",
	}
	c.Check(result, DeepEquals, expected)
}

func (s *PackageDependenciesSuite) TestParseDependenciesProfiles(c *C) {
	// Test parsing dependencies with build profiles
	stanza := Stanza{
		"Depends": "package1 <cross>, package2 <!nocheck>, package3 <stage1 !cross>",
	}

	result := parseDependencies(stanza, "Depends")
	expected := []string{
		"package1 <cross>",
		"package2 <!nocheck>",
		"package3 <stage1 !cross>",
	}
	c.Check(result, DeepEquals, expected)
}

func (s *PackageDependenciesSuite) TestParseDependenciesLongLine(c *C) {
	// Test parsing very long dependency line
	longDeps := "pkg1, pkg2, pkg3, pkg4, pkg5, pkg6, pkg7, pkg8, pkg9, pkg10, " +
		"pkg11, pkg12, pkg13, pkg14, pkg15, pkg16, pkg17, pkg18, pkg19, pkg20"

	stanza := Stanza{
		"Depends": longDeps,
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(len(result), Equals, 20)
	c.Check(result[0], Equals, "pkg1")
	c.Check(result[19], Equals, "pkg20")
}

func (s *PackageDependenciesSuite) TestParseDependenciesSingleComma(c *C) {
	// Test edge case with single comma
	stanza := Stanza{
		"Depends": ",",
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, DeepEquals, []string{"", ""})
}

func (s *PackageDependenciesSuite) TestParseDependenciesTrailingComma(c *C) {
	// Test with trailing comma
	stanza := Stanza{
		"Depends": "package1, package2,",
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, DeepEquals, []string{"package1", "package2", ""})
}

func (s *PackageDependenciesSuite) TestParseDependenciesLeadingComma(c *C) {
	// Test with leading comma
	stanza := Stanza{
		"Depends": ", package1, package2",
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, DeepEquals, []string{"", "package1", "package2"})
}

func (s *PackageDependenciesSuite) TestParseDependenciesMultipleCommas(c *C) {
	// Test with multiple consecutive commas
	stanza := Stanza{
		"Depends": "package1,, package2,,, package3",
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, DeepEquals, []string{"package1", "", "package2", "", "", "package3"})
}

func (s *PackageDependenciesSuite) TestParseDependenciesRealWorld(c *C) {
	// Test with real-world dependency examples
	stanza := Stanza{
		"Depends": "debconf (>= 0.5) | debconf-2.0, libc6 (>= 2.14), libgcc1 (>= 1:3.0), libstdc++6 (>= 5.2)",
	}

	result := parseDependencies(stanza, "Depends")
	expected := []string{
		"debconf (>= 0.5) | debconf-2.0",
		"libc6 (>= 2.14)",
		"libgcc1 (>= 1:3.0)",
		"libstdc++6 (>= 5.2)",
	}
	c.Check(result, DeepEquals, expected)
}

func (s *PackageDependenciesSuite) TestParseDependenciesDifferentKeys(c *C) {
	// Test parsing different dependency types
	stanza := Stanza{
		"Depends":             "runtime-dep",
		"Build-Depends":       "build-dep",
		"Build-Depends-Indep": "build-indep-dep",
		"Pre-Depends":         "pre-dep",
		"Suggests":            "suggest-dep",
		"Recommends":          "recommend-dep",
	}

	// Test each dependency type
	depends := parseDependencies(stanza, "Depends")
	c.Check(depends, DeepEquals, []string{"runtime-dep"})

	buildDepends := parseDependencies(stanza, "Build-Depends")
	c.Check(buildDepends, DeepEquals, []string{"build-dep"})

	buildDependsIndep := parseDependencies(stanza, "Build-Depends-Indep")
	c.Check(buildDependsIndep, DeepEquals, []string{"build-indep-dep"})

	preDepends := parseDependencies(stanza, "Pre-Depends")
	c.Check(preDepends, DeepEquals, []string{"pre-dep"})

	suggests := parseDependencies(stanza, "Suggests")
	c.Check(suggests, DeepEquals, []string{"suggest-dep"})

	recommends := parseDependencies(stanza, "Recommends")
	c.Check(recommends, DeepEquals, []string{"recommend-dep"})

	// Verify all keys were removed
	c.Check(len(stanza), Equals, 0)
}

func (s *PackageDependenciesSuite) TestPackageDependenciesStruct(c *C) {
	// Test PackageDependencies struct creation and field access
	deps := PackageDependencies{
		Depends:           []string{"dep1", "dep2"},
		BuildDepends:      []string{"build-dep1", "build-dep2"},
		BuildDependsInDep: []string{"build-indep-dep1"},
		PreDepends:        []string{"pre-dep1"},
		Suggests:          []string{"suggest1", "suggest2"},
		Recommends:        []string{"recommend1"},
	}

	c.Check(deps.Depends, DeepEquals, []string{"dep1", "dep2"})
	c.Check(deps.BuildDepends, DeepEquals, []string{"build-dep1", "build-dep2"})
	c.Check(deps.BuildDependsInDep, DeepEquals, []string{"build-indep-dep1"})
	c.Check(deps.PreDepends, DeepEquals, []string{"pre-dep1"})
	c.Check(deps.Suggests, DeepEquals, []string{"suggest1", "suggest2"})
	c.Check(deps.Recommends, DeepEquals, []string{"recommend1"})
}

func (s *PackageDependenciesSuite) TestParseDependenciesUnicodeCharacters(c *C) {
	// Test parsing dependencies with unicode characters
	stanza := Stanza{
		"Depends": "libμ-package, package-ñoño, 中文-package",
	}

	result := parseDependencies(stanza, "Depends")
	expected := []string{
		"libμ-package",
		"package-ñoño",
		"中文-package",
	}
	c.Check(result, DeepEquals, expected)
}

func (s *PackageDependenciesSuite) TestParseDependenciesStanzaImmutability(c *C) {
	// Test that original stanza values are not modified (except for key removal)
	original := Stanza{
		"Depends": "package1, package2",
		"Other":   "value",
	}

	// Make a copy to compare
	stanza := Stanza{
		"Depends": original["Depends"],
		"Other":   original["Other"],
	}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, DeepEquals, []string{"package1", "package2"})

	// Check that Depends key was removed but Other remains unchanged
	_, dependsExists := stanza["Depends"]
	c.Check(dependsExists, Equals, false)
	c.Check(stanza["Other"], Equals, original["Other"])
}

func (s *PackageDependenciesSuite) TestParseDependenciesEmptyStanza(c *C) {
	// Test with completely empty stanza
	stanza := Stanza{}

	result := parseDependencies(stanza, "Depends")
	c.Check(result, IsNil)
	c.Check(len(stanza), Equals, 0)
}

func (s *PackageDependenciesSuite) TestParseDependenciesTabsAndNewlines(c *C) {
	// Test parsing dependencies with tabs and newlines
	stanza := Stanza{
		"Depends": "package1,\n\tpackage2,\t package3\n,package4",
	}

	result := parseDependencies(stanza, "Depends")
	// The function should handle tabs and newlines as whitespace
	c.Check(len(result), Equals, 4)
	c.Check(result[0], Equals, "package1")
	c.Check(result[3], Equals, "package4")
}
