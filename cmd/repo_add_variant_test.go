package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	ctx "github.com/aptly-dev/aptly/context"
	"github.com/aptly-dev/aptly/deb"
	"github.com/aptly-dev/aptly/utils"
	"github.com/smira/flag"

	"gopkg.in/check.v1"
)

type RepoAddVariantSuite struct{}

var _ = check.Suite(&RepoAddVariantSuite{})

// Every fixture has the same uninformative basename. Architecture-Variant is
// supplied exclusively by control metadata, and rebuilt packages change payload.
func (*RepoAddVariantSuite) TestRepoAddArchitectureVariant(c *check.C) {
	builder, err := exec.LookPath("dpkg-deb")
	c.Assert(err, check.IsNil, check.Commentf("these real .deb import tests require dpkg-deb"))
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
		dir := c.MkDir()
		root := filepath.Join(dir, "root")
		c.Assert(os.MkdirAll(filepath.Join(root, "DEBIAN"), 0755), check.IsNil)
		control := fmt.Sprintf("Package: import-test\nVersion: %s\nArchitecture: amd64\nMaintainer: Test <test@example.invalid>\nDescription: tiny import fixture\n", spec.version)
		if spec.variant != "" {
			control += "Architecture-Variant: " + spec.variant + "\n"
		}
		c.Assert(os.WriteFile(filepath.Join(root, "DEBIAN", "control"), []byte(control), 0644), check.IsNil)
		c.Assert(os.WriteFile(filepath.Join(root, "payload"), []byte(spec.payload), 0644), check.IsNil)
		path := filepath.Join(dir, "payload.deb")
		output, err := exec.Command(builder, "--build", "--root-owner-group", root, path).CombinedOutput()
		c.Assert(err, check.IsNil, check.Commentf("build fixture: %s", output))
		checksums, err := utils.ChecksumsForFile(path)
		c.Assert(err, check.IsNil)
		pkg := &deb.Package{Name: "import-test", Version: spec.version, Architecture: "amd64", ArchitectureVariant: spec.variant, V06Plus: true}
		pkg.UpdateFiles(deb.PackageFiles{{Filename: "payload.deb", Checksums: checksums}})
		data, err := os.ReadFile(path)
		c.Assert(err, check.IsNil)
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
		c.Logf("case: %s", tc.name)
		func() {
			savedConfig, savedContext := utils.Config, context
			defer func() { utils.Config, context = savedConfig, savedContext }()
			dir := c.MkDir()
			configPath := filepath.Join(dir, "aptly.conf")
			config, err := json.Marshal(map[string]interface{}{"rootDir": dir, "gpgProvider": "internal"})
			c.Assert(err, check.IsNil)
			c.Assert(os.WriteFile(configPath, config, 0600), check.IsNil)
			flags := flag.NewFlagSet("repo-add-test", flag.ContinueOnError)
			flags.String("config", configPath, "")
			flags.Bool("no-lock", false, "")
			flags.Int("db-open-attempts", 1, "")
			flags.Bool("force-replace", false, "")
			flags.Bool("remove-files", false, "")
			testContext, err := ctx.NewContext(flags)
			c.Assert(err, check.IsNil)
			context = testContext
			defer testContext.Shutdown()
			factory := testContext.NewCollectionFactory()
			c.Assert(factory.LocalRepoCollection().Add(deb.NewLocalRepo("test", "")), check.IsNil)
			add := func(id string) error { return aptlyRepoAdd(makeCmdRepoAdd(), []string{"test", fixtures[id].path}) }
			for _, id := range tc.initial {
				c.Assert(add(id), check.IsNil)
			}
			c.Assert(flags.Set("force-replace", fmt.Sprint(tc.force)), check.IsNil)
			err = add(tc.incoming)
			c.Assert(err != nil, check.Equals, tc.reject, check.Commentf("case %s: repo add error: %v", tc.name, err))
			// Reload persisted references and metadata rather than checking an in-memory list.
			factory = testContext.NewCollectionFactory()
			repo, err := factory.LocalRepoCollection().ByName("test")
			c.Assert(err, check.IsNil)
			c.Assert(factory.LocalRepoCollection().LoadComplete(repo), check.IsNil)
			got := map[string]string{}
			for _, key := range repo.RefList().Refs {
				p, err := factory.PackageCollection().ByKey(key)
				c.Assert(err, check.IsNil)
				c.Check(p.Architecture, check.Equals, "amd64", check.Commentf("case %s", tc.name))
				c.Assert(p.Files(), check.HasLen, 1)
				got[string(p.ShortKey(""))] = p.Files()[0].Checksums.SHA256
			}
			want := map[string]string{}
			for _, id := range tc.want {
				p := fixtures[id].pkg
				want[string(p.ShortKey(""))] = p.Files()[0].Checksums.SHA256
			}
			c.Check(got, check.DeepEquals, want, check.Commentf("case %s: persisted package identities", tc.name))
			// Import stores files and full DB records before resolving repo conflicts.
			// Neither rejection nor replacement should corrupt/remove these pool blobs.
			for _, id := range append(append([]string{}, tc.initial...), tc.incoming) {
				f := fixtures[id]
				p, err := factory.PackageCollection().ByKey(f.pkg.Key(""))
				c.Assert(err, check.IsNil)
				c.Check(p.ArchitectureVariant, check.Equals, f.pkg.ArchitectureVariant, check.Commentf("case %s", tc.name))
				stream, err := testContext.PackagePool().Open(p.Files()[0].PoolPath)
				c.Assert(err, check.IsNil)
				data, err := io.ReadAll(stream)
				closeErr := stream.Close()
				c.Assert(err, check.IsNil)
				c.Assert(closeErr, check.IsNil)
				c.Check(data, check.DeepEquals, f.data, check.Commentf("case %s: pool content for %s", tc.name, id))
				_, err = os.Stat(f.path)
				c.Check(err, check.IsNil, check.Commentf("case %s: input file %s", tc.name, f.path))
			}
		}()
	}
}
