// This utility corrupts a database by deleting matching package entries.
// Do not use it outside system tests.
package main

import (
	"flag"
	"log"

	"github.com/aptly-dev/aptly/database/goleveldb"
)

func main() {
	var dbPath, prefix string

	flag.StringVar(&dbPath, "db", "", "Path to DB to corrupt")
	flag.StringVar(&prefix, "prefix", "P", "Path to DB to corrupt")
	flag.Parse()

	db, err := goleveldb.NewOpenDB(dbPath)
	if err != nil {
		log.Fatalf("Error opening DB %q: %s", dbPath, err)
	}
	defer db.Close()

	keys := db.KeysByPrefix([]byte(prefix))
	if len(keys) == 0 {
		keys2 := db.KeysByPrefix([]byte{})
		for _, key := range keys2 {
			log.Printf("Have: %q", key)
		}

		log.Fatal("No keys to delete")
	}

	for _, key := range keys {
		log.Printf("Deleting %q", key)

		if err = db.Delete(key); err != nil {
			log.Fatalf("Error deleting key: %s", err)
		}
	}
}
