package deb

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// ApplyFilter operates on the package list already parsed from downloaded
// indexes. All architecture information here comes from control metadata.
func TestRemoteRepoApplyFilterArchitectureVariant(t *testing.T) {
	binary := func(name, variant, arch, depends, source string) *Package {
		control := "Package: " + name + "\nVersion: 1\nArchitecture: " + arch + "\n"
		if variant != "" {
			control += "Architecture-Variant: " + variant + "\n"
		}
		if depends != "" {
			control += "Depends: " + depends + "\n"
		}
		if source != "" {
			control += "Source: " + source + "\n"
		}
		stanza, err := NewControlFileReader(strings.NewReader(control), false, false).ReadStanza()
		if err != nil {
			t.Fatal(err)
		}
		return NewPackageFromControlFile(stanza)
	}
	source, err := NewSourcePackageFromControlFile(Stanza{
		"Package": "source-app", "Version": "1", "Architecture": "any all", "Build-Depends": "build-tool",
	})
	if err != nil {
		t.Fatal(err)
	}
	appNormal := binary("app", "", "amd64", "helper", "")
	appVariant := binary("app", "amd64v3", "amd64", "helper", "")
	helperNormal := binary("helper", "", "amd64", "", "")
	helperVariant := binary("helper", "amd64v3", "amd64", "", "")
	helperAll := binary("helper", "", "all", "", "")
	appAll := binary("app", "", "all", "helper", "")
	sourceVariant := binary("app", "amd64v3", "amd64", "", "source-app")
	sourceNormal := binary("app", "", "amd64", "", "source-app")
	buildTool := binary("build-tool", "", "amd64", "", "")
	for _, tc := range []struct {
		name                  string
		indexes               []string
		input, want           []*Package
		queryName             string
		withDeps, withSources bool
		options               int
	}{
		{"variant_index_normal_dependency", []string{"amd64v3"}, []*Package{appVariant, helperNormal}, []*Package{appVariant, helperNormal}, "app", true, false, 0},
		{"variant_index_variant_dependency", []string{"amd64v3"}, []*Package{appVariant, helperVariant}, []*Package{appVariant, helperVariant}, "app", true, false, 0},
		{"variant_index_all_dependency", []string{"amd64v3"}, []*Package{appVariant, helperAll}, []*Package{appVariant, helperAll}, "app", true, false, 0},
		{"variant_index_all_selected", []string{"amd64v3"}, []*Package{appAll, helperVariant}, []*Package{appAll, helperVariant}, "app", true, false, 0},
		{"mixed_indexes_shared_dependency", []string{"amd64", "amd64v3"}, []*Package{appNormal, appVariant, helperNormal}, []*Package{appNormal, appVariant, helperNormal}, "app", true, false, 0},
		{"mixed_indexes_shared_all_dependency", []string{"amd64", "amd64v3"}, []*Package{appNormal, appVariant, helperAll}, []*Package{appNormal, appVariant, helperAll}, "app", true, false, 0},
		{"normal_index_binary_dependency", []string{"amd64"}, []*Package{appNormal, helperNormal}, []*Package{appNormal, helperNormal}, "app", true, false, 0},
		{"normal_index_all_dependency", []string{"amd64"}, []*Package{appNormal, helperAll}, []*Package{appNormal, helperAll}, "app", true, false, 0},
		{"dependencies_disabled", []string{"amd64v3"}, []*Package{appVariant, helperVariant}, []*Package{appVariant}, "app", false, false, 0},
		{"variant_follow_source", []string{"amd64v3"}, []*Package{sourceVariant, source}, []*Package{sourceVariant, source}, "app", true, false, DepFollowSource},
		{"normal_follow_source", []string{"amd64"}, []*Package{sourceNormal, source}, []*Package{sourceNormal, source}, "app", true, false, DepFollowSource},
		{"download_sources_explicit_name", []string{"amd64v3"}, []*Package{sourceVariant, source}, []*Package{sourceVariant, source}, "app", true, true, 0},
		// These protect existing behavior against blindly passing Architectures(false).
		{"only_all_normal_index", []string{"amd64"}, []*Package{appAll, helperAll}, []*Package{appAll, helperAll}, "app", true, false, 0},
		{"only_all_variant_index", []string{"amd64v3"}, []*Package{appAll, helperAll}, []*Package{appAll, helperAll}, "app", true, false, 0},
		{"source_only", []string{"amd64"}, []*Package{source}, []*Package{source}, "source-app", true, true, DepFollowBuild},
		// Mirrors do not currently walk source-package build dependencies. Retaining
		// that behavior avoids treating the source pseudo-architecture as a CPU arch.
		{"source_build_dependencies_unchanged", []string{"amd64"}, []*Package{source, buildTool}, []*Package{source}, "source-app", true, true, DepFollowBuild},
		{"empty_selection", []string{"amd64v3"}, []*Package{appVariant, helperVariant}, nil, "absent", true, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			list := NewPackageList()
			for _, p := range tc.input {
				if err := list.Add(p); err != nil {
					t.Fatal(err)
				}
			}
			bases := list.Architectures(false)
			sort.Strings(bases)
			t.Logf("index architectures=%v; metadata base architectures=%v", tc.indexes, bases)
			repo := &RemoteRepo{Architectures: append([]string{}, tc.indexes...), FilterWithDeps: tc.withDeps, DownloadSources: tc.withSources, packageList: list}
			oldLen, newLen, err := repo.ApplyFilter(tc.options, &FieldQuery{Field: "Name", Relation: VersionEqual, Value: tc.queryName}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if oldLen != len(tc.input) {
				t.Errorf("oldLen=%d, want %d", oldLen, len(tc.input))
			}
			got := repo.packageList.FullNames()
			want := make([]string, 0, len(tc.want))
			for _, p := range tc.want {
				want = append(want, p.GetFullName())
			}
			sort.Strings(got)
			sort.Strings(want)
			if newLen != len(tc.want) || !reflect.DeepEqual(got, want) {
				t.Errorf("filtered packages (newLen=%d):\n got %v\nwant %v", newLen, got, want)
			}
			if !reflect.DeepEqual(repo.Architectures, tc.indexes) {
				t.Errorf("index configuration changed: %v", repo.Architectures)
			}
		})
	}
}
