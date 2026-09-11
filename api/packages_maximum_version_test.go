package api

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"

	"github.com/aptly-dev/aptly/database/goleveldb"
	"github.com/aptly-dev/aptly/deb"
	"github.com/gin-gonic/gin"
)

// Exercise the common API listing handler with persisted packages, checking
// both reference and detailed JSON responses independently of response order.
func TestPackagesMaximumVersionArchitectureVariant(t *testing.T) {
	pkg := func(arch, variant, version string) *deb.Package {
		stanza := deb.Stanza{"Package": "example", "Version": version, "Architecture": arch}
		if variant != "" {
			stanza["Architecture-Variant"] = variant
		}
		return deb.NewPackageFromControlFile(stanza)
	}
	n1, n3 := pkg("amd64", "", "1"), pkg("amd64", "", "3")
	v1, v2 := pkg("amd64", "amd64v3", "1"), pkg("amd64", "amd64v3", "2")
	a1, a4 := pkg("arm64", "", "1"), pkg("arm64", "", "4")
	all := []*deb.Package{n1, n3, v1, v2, a1, a4}
	cases := []struct {
		name, option string
		input, want  []*deb.Package
	}{
		{"same_version", "1", []*deb.Package{n1, v1}, []*deb.Package{n1, v1}},
		{"independent_latest", "1", []*deb.Package{n1, n3, v1, v2}, []*deb.Package{n3, v2}},
		{"ordinary_multi_architecture", "1", []*deb.Package{n1, n3, a1, a4}, []*deb.Package{n3, a4}},
		{"ordinary_single_architecture", "1", []*deb.Package{n1, n3}, []*deb.Package{n3}},
		{"disabled_absent", "", all, all},
		{"disabled_zero", "0", all, all},
		{"empty", "1", nil, nil},
	}
	for _, tc := range cases {
		for _, format := range []string{"refs", "details"} {
			t.Run(tc.name+"/"+format, func(t *testing.T) {
				db, err := goleveldb.NewOpenDB(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := db.Close(); err != nil {
						t.Error(err)
					}
				})
				factory := deb.NewCollectionFactory(db)
				list := deb.NewPackageList()
				for _, p := range tc.input {
					// Update offloads metadata; use a fresh package for each database.
					p := pkg(p.Architecture, p.ArchitectureVariant, p.Version)
					if err := factory.PackageCollection().Update(p); err != nil {
						t.Fatal(err)
					}
					if err := list.Add(p); err != nil {
						t.Fatal(err)
					}
				}
				refs := deb.NewPackageRefListFromPackageList(list)
				url := "/api/packages?format=" + format
				if tc.option != "" {
					url += "&maximumVersion=" + tc.option
				}
				response := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(response)
				c.Request = httptest.NewRequest("GET", url, nil)
				showPackages(c, refs, deb.NewCollectionFactory(db))
				if response.Code != 200 {
					t.Fatalf("status %d: %s", response.Code, response.Body.String())
				}
				got := []string{}
				if format == "details" {
					var details []map[string]string
					if err := json.Unmarshal(response.Body.Bytes(), &details); err != nil {
						t.Fatal(err)
					}
					for _, detail := range details {
						got = append(got, detail["Key"])
						// Metadata must remain separate from the identity encoded in Key.
						for _, p := range tc.want {
							if detail["Key"] == string(p.Key("")) {
								if detail["Architecture"] != p.Architecture || detail["Architecture-Variant"] != p.ArchitectureVariant || detail["ShortKey"] != string(p.ShortKey("")) {
									t.Errorf("incorrect package metadata: %v", detail)
								}
							}
						}
					}
				} else if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				want := []string{}
				for _, p := range tc.want {
					want = append(want, string(p.Key("")))
				}
				sort.Strings(got)
				sort.Strings(want)
				if !reflect.DeepEqual(got, want) {
					t.Errorf("package references:\n got %v\nwant %v", got, want)
				}
				if !reflect.DeepEqual(refs.Strings(), deb.NewPackageRefListFromPackageList(list).Strings()) {
					t.Error("listing mutated source references")
				}
			})
		}
	}
}
