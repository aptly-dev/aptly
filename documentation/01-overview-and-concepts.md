# 1. Overview & Concepts

## 1.1 What problem aptly solves

`apt` clients consume *repositories*: a directory tree containing `dists/<distribution>/…`
index files (`Release`, `Packages`, `Sources`, `Contents`) and a `pool/` of package files.
Running one of those trees in production raises hard questions:

* How do I get a **partial copy** of `archive.ubuntu.com` (only some components/architectures,
  only some packages and their dependencies)?
* How do I make installation **repeatable** — i.e. freeze "what the archive looked like on
  Tuesday" and ship exactly that to 500 machines?
* How do I **publish my own packages** as a proper, signed repository?
* How do I **promote** a tested set of packages from *staging* to *production*, atomically,
  with the ability to roll back?

aptly answers with a small set of composable objects and operations. Its stated goal (from the
CLI help in `cmd/cmd.go`) is "to establish repeatability and controlled changes in a
package-centric environment".

## 1.2 The object model

```
                    (remote archive on the internet)
                                │  aptly mirror create / update
                                ▼
   ┌───────────────┐    snapshot     ┌────────────┐  filter / merge / pull  ┌────────────┐
   │  Mirror       │ ──────────────► │  Snapshot  │ ──────────────────────► │  Snapshot  │
   │  (RemoteRepo) │                 │ (immutable)│                         │ (immutable)│
   └───────────────┘                 └─────┬──────┘                         └─────┬──────┘
                                           │                                      │
   ┌───────────────┐    snapshot           │            aptly publish snapshot    │
   │  Local repo   │ ──────────────────────┘                     │                │
   │  (LocalRepo)  │◄── aptly repo add / include / import        ▼                ▼
   └───────┬───────┘                                    ┌────────────────────────────────┐
           │  aptly publish repo (direct)               │   Published repository         │
           └──────────────────────────────────────────► │   (PublishedRepo)              │
                                                        │  = dists/ + pool/ on a storage │
                                                        └────────────────────────────────┘
```

| Object | Go type | Mutable? | What it holds |
|--------|---------|----------|---------------|
| **Package** | `deb.Package` | no | One `.deb`/`.udeb`/source package: name, version, architecture, dependencies, extra control fields, list of files with checksums |
| **Mirror** | `deb.RemoteRepo` | yes (via `update`) | Configuration of a remote archive (URL, distribution, components, architectures, filter) + the list of packages found at last update |
| **Local repository** | `deb.LocalRepo` | yes | A user-managed list of packages (added from files, or copied from other places) |
| **Snapshot** | `deb.Snapshot` | **no** | An immutable list of packages + provenance ("created from mirror X", "merge of A and B", …) |
| **Published repository** | `deb.PublishedRepo` | re-publishable | A binding *(storage, prefix, distribution)* → one source per *component* (snapshot or local repo), plus the generated files |
| **Package pool** | `aptly.PackagePool` | append/GC | De-duplicated storage of the actual package files |
| **Published storage** | `aptly.PublishedStorage` | | Where the published tree lives (directory, S3, GCS, Azure, Swift, JFrog) |

Two crucial facts drive everything else:

1. **Repositories, mirrors and snapshots do not contain packages — they contain *references*
   to packages.** A reference is a short byte string (`"Pamd64 nginx 1.24.0-1 8f3a91c2"`);
   the package record itself lives once in the database. Creating a snapshot of a
   100,000-package mirror therefore copies a sorted list of 100,000 short keys, not
   100,000 packages (see [chapter 3](03-data-model-and-database.md)).
2. **Package *files* are stored once**, in a pool addressed by SHA-256, no matter how many
   mirrors/snapshots/publications use them (see [chapter 4](04-storage.md)).

## 1.3 Life cycle of a package

1. **Discovered.** Either parsed from a remote `Packages`/`Sources` index (mirror update) or
   read from a `.deb`/`.dsc` on disk (`repo add`, `repo include`).
2. **Registered.** A `deb.Package` is written to the database under a key derived from
   *(architecture, name, version, hash-of-files)*. Large/rarely used fields (file list,
   dependency lists, extra control fields) are stored under separate "offload" keys.
3. **Downloaded/imported.** The package file(s) are downloaded (mirror) or copied (local repo)
   into the pool at a path derived from the file's SHA-256.
4. **Referenced.** The package key is inserted into a `PackageRefList` belonging to a mirror,
   local repo or snapshot.
5. **Published.** When a repository containing it is published, the file is hard-linked
   (or symlinked/copied/uploaded) from the pool into `pool/<component>/<x>/<source>/…` on the
   published storage, and a stanza describing it is written to `Packages`.
6. **Garbage collected.** After all references disappear (mirror dropped, snapshot deleted,
   …) `aptly db cleanup` removes the package record and — if no other package references it —
   the pool file.

## 1.4 Typical workflows

### A. Mirror upstream, freeze, publish

```bash
aptly mirror create -architectures=amd64 -filter='nginx | curl' -filter-with-deps \
      bookworm-main http://deb.debian.org/debian bookworm main
aptly mirror update bookworm-main                    # download indexes + packages
aptly snapshot create bookworm-2026-09-30 from mirror bookworm-main
aptly publish snapshot -distribution=bookworm bookworm-2026-09-30 debian
# clients: deb http://host/debian bookworm main
```

### B. Publish your own packages

```bash
aptly repo create -distribution=stable -component=main myrepo
aptly repo add myrepo ./build/*.deb                  # or: aptly repo include ./build/x.changes
aptly publish repo myrepo internal                   # publishes directly from the local repo
aptly repo add myrepo ./build/newer.deb
aptly publish update stable internal                 # re-publish (only changed state)
```

### C. Controlled upgrade (`pull`) and atomic switch

```bash
# take one package (+dependencies) from a newer snapshot into the current one
aptly snapshot pull -no-remove prod-2026-09-01 bookworm-2026-09-30 prod-2026-10-01 nginx
aptly publish switch bookworm prod-2026-10-01 debian   # atomically re-point the publication
# rollback: aptly publish switch bookworm prod-2026-09-01 debian
```

### D. Multi-component / multi-source publishing

```bash
aptly publish snapshot -component=main,contrib -distribution=stable snap-main snap-contrib
```

Each component of a published repository is fed by its *own* source (snapshot or local repo).
All components of one publication must have the same source kind.

### E. REST-driven automation (CI/CD)

```bash
curl -F file=@app_1.0_amd64.deb http://aptly:8080/api/files/ci-upload
curl -X POST http://aptly:8080/api/repos/myrepo/file/ci-upload
curl -X PUT  -H 'Content-Type: application/json' \
     -d '{"Signing":{"Passphrase":"…"}}' \
     http://aptly:8080/api/publish/internal/stable
```

## 1.5 What aptly deliberately does *not* do

* It is **not a package build system** (it only ingests finished `.deb`/`.dsc`/`.changes`).
* It is **not a long-running daemon by default**: the CLI is a batch tool; `aptly api serve`
  is the (optional) server mode.
* It does **not** support RPM/YUM repositories (the `aptly` Go package comment mentions
  "doesn't depend directly on Debian or CentOS", but only Debian formats are implemented).
* It is **not multi-writer**: with the default LevelDB backend only one process may open the
  database at a time (see [Operations](12-operations.md)). The `etcd` backend and the
  server's `-no-lock` mode relax this.
