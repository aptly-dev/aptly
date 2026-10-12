package query

import (
	"reflect"
	"sort"
	"testing"

	"github.com/aptly-dev/aptly/database/goleveldb"
	"github.com/aptly-dev/aptly/deb"
)

func TestArchitectureVariantQueries(t *testing.T) {
	db, err := goleveldb.NewOpenDB(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}()
	collection := deb.NewPackageCollection(db)
	list := deb.NewPackageList()
	const normal = "test-package_1.0_amd64"
	const variant = "test-package_1.0_amd64v3"
	for _, value := range []string{"", "amd64v3"} {
		p := deb.NewPackageFromControlFile(deb.Stanza{
			"Package": "test-package", "Version": "1.0",
			"Architecture": "amd64", "Architecture-Variant": value,
		})
		if err := collection.Update(p); err != nil {
			t.Fatal(err)
		}
		if err := list.Add(p); err != nil {
			t.Fatal(err)
		}
	}
	list.PrepareIndex()
	for _, catalog := range []struct {
		name  string
		value deb.PackageCatalog
	}{{"list", list}, {"collection", collection}} {
		t.Run(catalog.name, func(t *testing.T) {
			for _, tc := range []struct {
				name, expression string
				want             []string
			}{
				{"coexistence", "Name (= test-package)", []string{normal, variant}},
				{"exact_normal", normal, []string{normal}},
				{"exact_variant", variant, []string{variant}},
				{"base_architecture", "Architecture (= amd64)", []string{normal, variant}},
				{"variant_is_not_base_architecture", "Architecture (= amd64v3)", []string{}},
				{"variant_field", "Architecture-Variant (= amd64v3)", []string{variant}},
				{"special_base_architecture", "$Architecture (= amd64)", []string{normal, variant}},
				{"special_variant_architecture", "$Architecture (= amd64v3)", []string{}},
				{"normal_and_field", normal + ", Architecture (= amd64)", []string{normal}},
				{"variant_and_field", variant + ", Architecture (= amd64)", []string{variant}},
				// A field OR operand forces scanning, including PkgQuery.Matches.
				{"scan_normal", normal + " | Name (= absent)", []string{normal}},
				{"scan_variant", variant + " | Name (= absent)", []string{variant}},
				{"not_normal", "!(" + normal + ")", []string{variant}},
				{"not_variant", "!(" + variant + ")", []string{normal}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					q, err := Parse(tc.expression)
					if err != nil {
						t.Fatal(err)
					}
					got := q.Query(catalog.value).FullNames()
					sort.Strings(got)
					if !reflect.DeepEqual(got, tc.want) {
						t.Errorf("query %q: got %v; want %v", tc.expression, got, tc.want)
					}
				})
			}
		})
	}
}

func TestArchitectureVariantQueryParsing(t *testing.T) {
	for _, arch := range []string{"amd64", "amd64v3"} {
		q, err := Parse("test-package_1.0_" + arch)
		if err != nil {
			t.Fatal(err)
		}
		want := &deb.PkgQuery{Pkg: "test-package", Version: "1.0", Arch: arch}
		if !reflect.DeepEqual(q, want) {
			t.Errorf("got %#v; want %#v", q, want)
		}
	}
}

func TestArchitectureVariantArchitectureAllQuerySemantics(t *testing.T) {
	p := &deb.Package{Name: "data", Architecture: "all"}
	for _, tc := range []struct {
		expression string
		want       bool
	}{
		{"Architecture (= amd64)", false},
		{"Architecture (= amd64v3)", false},
		{"$Architecture (= amd64)", true},
		{"$Architecture (= amd64v3)", true},
		{"$Architecture (= source)", false},
	} {
		q, err := Parse(tc.expression)
		if err != nil {
			t.Fatal(err)
		}
		if got := q.Matches(p); got != tc.want {
			t.Errorf("query %q: got %v; want %v", tc.expression, got, tc.want)
		}
	}
}
