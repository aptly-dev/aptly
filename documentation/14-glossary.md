# 14. Glossary

| Term | Meaning in aptly |
|------|------------------|
| **Acquire-By-Hash** | apt feature: index files fetched from `by-hash/<HASH>/<digest>` so they're immutable; enabled with `-acquire-by-hash` |
| **Architecture** | `amd64`, `arm64`, `all` (arch-independent), `source`, … Package files of arch `all` are published into every `binary-<arch>` index |
| **Batch** | `database.Batch`: several writes applied atomically |
| **Checksum cache** | DB keys `C<poolPath>` remembering `ChecksumInfo` of pool files |
| **Collection** | Repository object (`RemoteRepoCollection`, `SnapshotCollection`, …) providing CRUD over one entity kind; obtained via `CollectionFactory` |
| **Component** | Subdivision of a distribution (`main`, `contrib`, `non-free`, …). Each published component has its own source |
| **Contents index** | `Contents-<arch>[.gz]` mapping file paths to packages (for `apt-file`) |
| **Control file / Stanza** | RFC-822-like `Field: value` block describing a package/release (`deb.Stanza`, `ControlFileReader`) |
| **Dangling reference** | Ref in a ref list whose package record no longer exists |
| **Distribution** | Release name (`bookworm`, `stable`); directory `dists/<distribution>` |
| **Endpoint** | Named published-storage target from the config (`s3:prod`, `filesystem:web1`) |
| **Flat repository** | Archive without `dists/`/components: `Packages` next to the debs; distribution written `./` or `sub/` |
| **Installer package** | Synthetic package representing debian-installer images in a mirror (`-with-installer`) |
| **InRelease / Release.gpg** | Clearsigned / detached-signed forms of `Release` |
| **Key (package key)** | `P<arch> <name> <version> <hash>`; unique identity of a package record |
| **Legacy pool** | Pre-1.1 MD5-based pool layout, read-only support |
| **Local repository** | User-managed list of packages (`deb.LocalRepo`) |
| **Mirror** | Configuration + snapshot-in-time list of a remote archive (`deb.RemoteRepo`) |
| **MultiDist** | Publication mode with per-distribution pool directories |
| **Offloaded fields** | Package data (files, deps, extra, contents) stored in separate keys `xF/xD/xE/xC` |
| **Package pool** | Content-addressed store for package files |
| **PackageList** | In-memory set of `Package`s, indexable and searchable |
| **PackageRefList** | Sorted list of package keys; the "contents" of mirror/repo/snapshot |
| **Prefix** | Path under a published storage where a publication lives (`.` = root) |
| **Progress** | Abstraction for progress bars and messages (`console.Progress`, `task.Output`) |
| **Publication / Published repo** | Binding (storage, prefix, distribution) → sources per component, plus generated files |
| **Pull** | Copy packages (+dependencies) from one snapshot into another, creating a new snapshot |
| **Query** | Package selection expression (see chapter 11) |
| **Resource (task)** | Key naming an object a task needs exclusively (`L<uuid>`, `__all__`, …) |
| **Revision (publish)** | Staged, not yet applied set of component→source changes on a publication |
| **rePublishing** | Internal flag: regeneration of an existing publication (uses `.tmp` names + rename) |
| **rootDir** | Base directory for db, pool, public, upload, skel |
| **Skeleton files** | Extra files under `rootDir/skel/<prefix>/dists/<dist>/<component>/` copied into the published component |
| **Snapshot** | Immutable ref list with provenance; source for publishing |
| **Source package** | Debian source (`.dsc` + tarballs); architecture `source` |
| **Stanza** | See *Control file* |
| **Switch** | Re-point a snapshot publication to different snapshots and regenerate |
| **Task** | Unit of work in the API's task list, with state, output and resource locks |
| **trustedkeys.gpg** | Keyring aptly's external verifier (`gpgv`) uses for remote repositories |
| **udeb** | Micro-package used by debian-installer (`-with-udebs`) |
| **Uploaders** | ACL (`uploaders.json`) governing which signing keys may `repo include` which packages |
| **Upload directory** | `rootDir/upload/<dir>`: staging area for API uploads |
| **UUID** | Random identifier of a persisted entity; basis for DB keys |
