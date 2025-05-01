package deb_test

import (
	"bytes"
	"testing"

	"github.com/aptly-dev/aptly/database/goleveldb"
	"github.com/aptly-dev/aptly/deb"
)

func TestFindDanglingReferences(t *testing.T) {
	reflist := deb.NewPackageRefList()
	reflist.Refs = [][]byte{[]byte("P existing 1.2.3"), []byte("P dangling 1.2.3")}

	db, _ := goleveldb.NewOpenDB(t.TempDir())
	packages := deb.NewPackageCollection(db)

	if err := packages.Update(&deb.Package{Name: "existing", Version: "1.2.3"}); err != nil {
		t.Fatal(err)
	}

	dangling, err := deb.FindDanglingReferences(reflist, packages)
	if err != nil {
		t.Fatal(err)
	}

	exp := &deb.PackageRefList{
		Refs: [][]byte{[]byte("P dangling 1.2.3")},
	}

	compareRefs(t, exp, dangling)
}

func compareRefs(t *testing.T, exp, got *deb.PackageRefList) {
	t.Helper()

	if len(exp.Refs) != len(got.Refs) {
		t.Fatalf("refs length mismatch: exp %d, got %d", len(exp.Refs), len(got.Refs))
	}

	for i := range exp.Refs {
		if !bytes.Equal(exp.Refs[i], got.Refs[i]) {
			t.Fatalf("refs do not match: exp %q, got %q", exp.Refs[i], got.Refs[i])
		}
	}
}
