package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	ctx "github.com/aptly-dev/aptly/context"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/utils"
	"github.com/smira/flag"
)

// Exercise the real pull entry point and persisted references, without package
// files or mirrors. Architecture selection remains base-architecture selection.
func TestSnapshotPullArchitectureVariant(t *testing.T) {
	normal := &deb.Package{Name: "pull-test", Version: "1", Architecture: "amd64"}
	variant := &deb.Package{Name: "pull-test", Version: "1", Architecture: "amd64", ArchitectureVariant: "amd64v3"}
	normal2 := &deb.Package{Name: "pull-test", Version: "2", Architecture: "amd64"}
	variant2 := &deb.Package{Name: "pull-test", Version: "2", Architecture: "amd64", ArchitectureVariant: "amd64v3"}
	cases := []struct {
		name                 string
		dest, source, want   []*deb.Package
		noRemove, allMatches bool
	}{
		{"normal", nil, []*deb.Package{normal}, []*deb.Package{normal}, false, false},
		{"variant", nil, []*deb.Package{variant}, []*deb.Package{variant}, false, false},
		{"both", nil, []*deb.Package{normal, variant}, []*deb.Package{normal, variant}, false, false},
		{"variant_into_normal", []*deb.Package{normal}, []*deb.Package{variant}, []*deb.Package{normal, variant}, false, false},
		{"normal_into_variant", []*deb.Package{variant}, []*deb.Package{normal}, []*deb.Package{normal, variant}, false, false},
		{"both_all_matches", nil, []*deb.Package{normal, variant}, []*deb.Package{normal, variant}, false, true},
		{"both_no_remove", nil, []*deb.Package{normal, variant}, []*deb.Package{normal, variant}, true, false},
		{"variant_into_normal_no_remove", []*deb.Package{normal}, []*deb.Package{variant}, []*deb.Package{normal, variant}, true, false},
		{"normal_into_variant_no_remove", []*deb.Package{variant}, []*deb.Package{normal}, []*deb.Package{normal, variant}, true, false},
		{"variant_into_normal_all_matches", []*deb.Package{normal}, []*deb.Package{variant}, []*deb.Package{normal, variant}, false, true},
		{"normal_into_variant_all_matches", []*deb.Package{variant}, []*deb.Package{normal}, []*deb.Package{normal, variant}, false, true},
		{"normal_replace", []*deb.Package{normal}, []*deb.Package{normal2}, []*deb.Package{normal2}, false, false},
		{"normal_no_remove", []*deb.Package{normal}, []*deb.Package{normal2}, []*deb.Package{normal, normal2}, true, false},
		{"normal_latest", nil, []*deb.Package{normal, normal2}, []*deb.Package{normal2}, false, false},
		{"normal_all_matches", nil, []*deb.Package{normal, normal2}, []*deb.Package{normal, normal2}, false, true},
		{"variant_replace_preserves_normal", []*deb.Package{normal, variant}, []*deb.Package{variant2}, []*deb.Package{normal, variant2}, false, false},
		{"latest_per_identity", nil, []*deb.Package{normal, normal2, variant, variant2}, []*deb.Package{normal2, variant2}, false, false},
		{"all_versions_both_identities", nil, []*deb.Package{normal, normal2, variant, variant2}, []*deb.Package{normal, normal2, variant, variant2}, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			savedConfig := utils.Config
			t.Cleanup(func() { utils.Config = savedConfig })
			dir := t.TempDir()
			configPath := filepath.Join(dir, "aptly.conf")
			configData, err := json.Marshal(map[string]interface{}{"rootDir": dir, "architectures": []string{}})
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(configPath, configData, 0600); err != nil {
				t.Fatal(err)
			}
			flags := flag.NewFlagSet("pull-test", flag.ContinueOnError)
			flags.String("config", configPath, "")
			// Empty destinations need explicit architecture; populated destinations
			// exercise automatic base-architecture discovery.
			arch := ""
			if len(tc.dest) == 0 {
				arch = "amd64"
			}
			flags.String("architectures", arch, "")
			flags.Bool("no-lock", false, "")
			flags.Int("db-open-attempts", 1, "")
			flags.Bool("no-deps", true, "")
			flags.Bool("no-remove", tc.noRemove, "")
			flags.Bool("all-matches", tc.allMatches, "")
			flags.Bool("dry-run", false, "")
			testContext, err := ctx.NewContext(flags)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(testContext.Shutdown)
			factory := testContext.NewCollectionFactory()
			refs := func(packages []*deb.Package) *deb.PackageRefList {
				list := deb.NewPackageList()
				for _, p := range packages {
					if err := list.Add(p); err != nil {
						t.Fatal(err)
					}
				}
				return deb.NewPackageRefListFromPackageList(list)
			}
			for _, p := range append(append([]*deb.Package{}, tc.dest...), tc.source...) {
				if err := factory.PackageCollection().Update(p); err != nil {
					t.Fatal(err)
				}
			}
			for name, packages := range map[string][]*deb.Package{"base": tc.dest, "source": tc.source} {
				if err := factory.SnapshotCollection().Add(deb.NewSnapshotFromRefList(name, nil, refs(packages), "")); err != nil {
					t.Fatal(err)
				}
			}
			savedContext := context
			context = testContext
			t.Cleanup(func() { context = savedContext })
			if err = aptlySnapshotPull(makeCmdSnapshotPull(), []string{"base", "source", "result", "pull-test"}); err != nil {
				t.Fatal(err)
			}
			snapshots := testContext.NewCollectionFactory().SnapshotCollection()
			result, err := snapshots.ByName("result")
			if err != nil {
				t.Fatal(err)
			}
			if err = snapshots.LoadComplete(result); err != nil {
				t.Fatal(err)
			}
			got, want := result.RefList().Strings(), refs(tc.want).Strings()
			if !reflect.DeepEqual(got, want) {
				t.Errorf("persisted package identities:\n got %v\nwant %v", got, want)
			}
		})
	}
}
