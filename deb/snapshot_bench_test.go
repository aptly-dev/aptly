package deb

import (
	"fmt"
	"os"
	"testing"

	"github.com/aptly-dev/aptly/database"
)

func BenchmarkSnapshotCollectionForEach(b *testing.B) {
	const count = 1024

	tmpDir := os.TempDir()
	defer os.RemoveAll(tmpDir)

	db, _ := database.NewOpenDB(tmpDir)
	defer db.Close()

	collection := NewSnapshotCollection(db)

	for i := 0; i < count; i++ {
		snapshot := NewSnapshotFromRefList(fmt.Sprintf("snapshot%d", i), nil, NewPackageRefList(), fmt.Sprintf("Snapshot number %d", i))
		if collection.Add(snapshot) != nil {
			b.FailNow()
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		collection = NewSnapshotCollection(db)

		collection.ForEach(func(s *Snapshot) error {
			return nil
		})
	}
}
