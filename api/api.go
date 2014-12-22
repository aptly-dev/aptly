// Package api provides implementation of aptly REST API
package api

// Lock order acquisition (canonical):
//  1. RemoteRepoCollection
//  2. LocalRepoCollection
//  3. SnapshotCollection
//  4. PublishedRepoCollection
