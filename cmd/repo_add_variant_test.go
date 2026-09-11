package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	ctx "github.com/aptly-dev/aptly/context"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/utils"
	"github.com/smira/flag"
)

// Every fixture has the same uninformative basename. Architecture-Variant is
// supplied exclusively by control metadata, and rebuilt packages change payload.
func TestRepoAddArchitectureVariant(t *testing.T) {
	builder, err := exec.LookPath("dpkg-deb")
	if err != nil {
		t.Fatal("these real .deb import tests require dpkg-deb:", err)
	}
	type fixture struct {
		path string
		pkg  *deb.Package
		data []byte
	}
	fixtures := map[string]fixture{}
	for _, spec := range []struct{ id, variant, version, payload string }{
		{"normal", "", "1", "original"}, {"variant", "amd64v3", "1", "original"},
		{"normal-rebuilt", "", "1", "rebuilt"}, {"variant-rebuilt", "amd64v3", "1", "rebuilt"},
		{"normal-v2", "", "2", "new version"}, {"variant-v2", "amd64v3", "2", "new version"},
	} {
		dir := t.TempDir()
		root := filepath.Join(dir, "root")
		if err := os.MkdirAll(filepath.Join(root, "DEBIAN"), 0755); err != nil {
			t.Fatal(err)
		}
		control := fmt.Sprintf("Package: import-test\nVersion: %s\nArchitecture: amd64\nMaintainer: Test <test@example.invalid>\nDescription: tiny import fixture\n", spec.version)
		if spec.variant != "" {
			control += "Architecture-Variant: " + spec.variant + "\n"
		}
		if err := os.WriteFile(filepath.Join(root, "DEBIAN", "control"), []byte(control), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "payload"), []byte(spec.payload), 0644); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "payload.deb")
		if output, err := exec.Command(builder, "--build", "--root-owner-group", root, path).CombinedOutput(); err != nil {
			t.Fatalf("build fixture: %v\n%s", err, output)
		}
		checksums, err := utils.ChecksumsForFile(path)
		if err != nil {
			t.Fatal(err)
		}
		pkg := &deb.Package{Name: "import-test", Version: spec.version, Architecture: "amd64", ArchitectureVariant: spec.variant, V06Plus: true}
		pkg.UpdateFiles(deb.PackageFiles{{Filename: "payload.deb", Checksums: checksums}})
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		fixtures[spec.id] = fixture{path, pkg, data}
	}
	for _, tc := range []struct {
		name          string
		initial       []string
		incoming      string
		force, reject bool
		want          []string
	}{
		{"normal_empty", nil, "normal", false, false, []string{"normal"}},
		{"variant_empty", nil, "variant", false, false, []string{"variant"}},
		{"normal_then_variant", []string{"normal"}, "variant", false, false, []string{"normal", "variant"}},
		{"variant_then_normal", []string{"variant"}, "normal", false, false, []string{"normal", "variant"}},
		{"identical_normal", []string{"normal", "variant"}, "normal", false, false, []string{"normal", "variant"}},
		{"identical_variant", []string{"normal", "variant"}, "variant", false, false, []string{"normal", "variant"}},
		{"conflicting_normal_without_force", []string{"normal", "variant"}, "normal-rebuilt", false, true, []string{"normal", "variant"}},
		{"conflicting_variant_without_force", []string{"normal", "variant"}, "variant-rebuilt", false, true, []string{"normal", "variant"}},
		{"force_normal_preserves_variant", []string{"normal", "variant"}, "normal-rebuilt", true, false, []string{"normal-rebuilt", "variant"}},
		{"force_variant_preserves_normal", []string{"normal", "variant"}, "variant-rebuilt", true, false, []string{"normal", "variant-rebuilt"}},
		{"force_identical_normal", []string{"normal", "variant"}, "normal", true, false, []string{"normal", "variant"}},
		{"force_identical_variant", []string{"normal", "variant"}, "variant", true, false, []string{"normal", "variant"}},
		{"force_new_variant_family", []string{"normal"}, "variant", true, false, []string{"normal", "variant"}},
		{"force_new_normal_family", []string{"variant"}, "normal", true, false, []string{"normal", "variant"}},
		{"force_normal_only", []string{"normal"}, "normal-rebuilt", true, false, []string{"normal-rebuilt"}},
		{"force_variant_only", []string{"variant"}, "variant-rebuilt", true, false, []string{"variant-rebuilt"}},
		{"force_normal_new_version", []string{"normal", "variant"}, "normal-v2", true, false, []string{"normal", "variant", "normal-v2"}},
		{"force_variant_new_version", []string{"normal", "variant"}, "variant-v2", true, false, []string{"normal", "variant", "variant-v2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			savedConfig, savedContext := utils.Config, context
			t.Cleanup(func() { utils.Config, context = savedConfig, savedContext })
			dir := t.TempDir()
			configPath := filepath.Join(dir, "aptly.conf")
			config, err := json.Marshal(map[string]interface{}{"rootDir": dir, "gpgProvider": "internal"})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, config, 0600); err != nil {
				t.Fatal(err)
			}
			flags := flag.NewFlagSet("repo-add-test", flag.ContinueOnError)
			flags.String("config", configPath, "")
			flags.Bool("no-lock", false, "")
			flags.Int("db-open-attempts", 1, "")
			flags.Bool("force-replace", false, "")
			flags.Bool("remove-files", false, "")
			testContext, err := ctx.NewContext(flags)
			if err != nil {
				t.Fatal(err)
			}
			context = testContext
			t.Cleanup(testContext.Shutdown)
			factory := testContext.NewCollectionFactory()
			if err := factory.LocalRepoCollection().Add(deb.NewLocalRepo("test", "")); err != nil {
				t.Fatal(err)
			}
			add := func(id string) error { return aptlyRepoAdd(makeCmdRepoAdd(), []string{"test", fixtures[id].path}) }
			for _, id := range tc.initial {
				if err := add(id); err != nil {
					t.Fatal(err)
				}
			}
			if err := flags.Set("force-replace", fmt.Sprint(tc.force)); err != nil {
				t.Fatal(err)
			}
			err = add(tc.incoming)
			if (err != nil) != tc.reject {
				t.Fatalf("repo add error: %v; want rejection %v", err, tc.reject)
			}
			// Reload persisted references and metadata rather than checking an in-memory list.
			factory = testContext.NewCollectionFactory()
			repo, err := factory.LocalRepoCollection().ByName("test")
			if err != nil {
				t.Fatal(err)
			}
			if err := factory.LocalRepoCollection().LoadComplete(repo); err != nil {
				t.Fatal(err)
			}
			got := map[string]string{}
			for _, key := range repo.RefList().Refs {
				p, err := factory.PackageCollection().ByKey(key)
				if err != nil {
					t.Fatal(err)
				}
				if p.Architecture != "amd64" {
					t.Errorf("base architecture changed: %s", p.Architecture)
				}
				if len(p.Files()) != 1 {
					t.Fatalf("package files: %v", p.Files())
				}
				got[string(p.ShortKey(""))] = p.Files()[0].Checksums.SHA256
			}
			want := map[string]string{}
			for _, id := range tc.want {
				p := fixtures[id].pkg
				want[string(p.ShortKey(""))] = p.Files()[0].Checksums.SHA256
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("persisted identities and contents:\n got %v\nwant %v", got, want)
			}
			// Import stores files and full DB records before resolving repo conflicts.
			// Neither rejection nor replacement should corrupt/remove these pool blobs.
			for _, id := range append(append([]string{}, tc.initial...), tc.incoming) {
				f := fixtures[id]
				p, err := factory.PackageCollection().ByKey(f.pkg.Key(""))
				if err != nil {
					t.Fatal(err)
				}
				if p.ArchitectureVariant != f.pkg.ArchitectureVariant {
					t.Errorf("metadata variant: got %q want %q", p.ArchitectureVariant, f.pkg.ArchitectureVariant)
				}
				stream, err := testContext.PackagePool().Open(p.Files()[0].PoolPath)
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(stream)
				closeErr := stream.Close()
				if err != nil {
					t.Fatal(err)
				}
				if closeErr != nil {
					t.Fatal(closeErr)
				}
				if !bytes.Equal(data, f.data) {
					t.Errorf("pool content changed for %s", id)
				}
				if _, err := os.Stat(f.path); err != nil {
					t.Errorf("input file lost: %v", err)
				}
			}
		})
	}
}
