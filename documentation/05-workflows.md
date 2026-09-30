# 5. Core Workflows (Step-by-Step Internals)

Each section follows the actual call sequence in the code, naming files and functions.
CLI entry points live in `cmd/`; REST entry points in `api/` reuse the same `deb` methods.

Contents:
[5.1 Mirror create & update](#51-mirror-update) ·
[5.2 Local repo: add / include / copy / move](#52-local-repositories) ·
[5.3 Snapshots: create / filter / merge / pull / diff / verify](#53-snapshots) ·
[5.4 Publish snapshot or repo](#54-publish) ·
[5.5 Switch / update / drop](#55-switch-update-drop) ·
[5.6 DB cleanup](#56-db-cleanup) ·
[5.7 Other commands](#57-other-commands)

---

## 5.1 Mirror update

### 5.1.1 `aptly mirror create <name> <url> <dist> [components…]` (`cmd/mirror_create.go`)

1. Parse `ppa:user/project` shorthand via `deb.ParsePPA` (uses `ppa_distributor_id`,
   `ppa_codename`, `ppa_baseurl`), otherwise take URL/distribution/components from args.
2. `deb.NewRemoteRepo(...)` — assigns a UUID, normalises the URL (`prepare()` adds a trailing
   `/`), detects **flat repositories** (distribution ends with `/` or starts with `.`), and
   rejects incompatible options (components/udebs/AppStream with flat repos).
3. Copy filter, `-filter-with-deps`, `-force-components`, `-force-architectures`; a filter
   string is **parsed immediately** with `query.Parse` so syntax errors surface at creation.
4. `repo.Fetch(downloader, verifier, ignoreSignatures)` — see below. This validates
   architectures/components against the archive's `Release`.
5. Save via `RemoteRepoCollection.Add` (no package list yet).

### 5.1.2 `RemoteRepo.Fetch` — obtain and verify `Release`

```
if ignoreSignatures:
     download Release ─ok→ use it
                     └fail→ download InRelease, strip clear-signature (no verification)
else:
     1. download InRelease; VerifyClearsigned; extract text        ── ok → done
     2. else download Release + Release.gpg; VerifyDetachedSignature ── ok → done
```

Then it parses the first stanza of the release:

* `Architectures:` → sorted; `source` removed. If the mirror has no architectures configured,
  all are adopted; otherwise each requested architecture must be a subset (error suggests
  `-force-architectures`).
* `Components:` (non-flat): for distributions with a path (`updates/main`) the leading
  `dist/` is stripped from component names. Same subset logic (`-force-components`).
* All of `MD5Sum`, `SHA1`, `SHA256`, `SHA512` blocks are parsed into
  `repo.ReleaseFiles[path] = ChecksumInfo{Size, hashes…}` — the **expected checksums for every
  index file**, later used to validate `Packages*.{bz2,gz,xz}` downloads.
* Remaining fields become `repo.Meta` (`Origin`, `Label`, `Suite`, `Codename`, `NotAutomatic`,
  `ButAutomaticUpgrades`, …), which snapshots inherit.

### 5.1.3 `aptly mirror update <name>` (`cmd/mirror_update.go`)

```mermaid
sequenceDiagram
    autonumber
    participant CLI as aptly mirror update
    participant R as RemoteRepo
    participant DL as Downloader
    participant DB as LevelDB
    participant P as PackagePool

    CLI->>DB: load mirror (LoadComplete → old ref list)
    CLI->>R: CheckLock() (unless -force)
    CLI->>DL: Fetch Release (verify signature)
    CLI->>DL: DownloadPackageIndexes (Packages/Sources/…)
    Note over R: parse stanzas → Package objects → in-memory PackageList
    opt AppStream enabled
        CLI->>DL: DownloadAppStreamFiles → pool
    end
    opt filter set
        CLI->>R: ApplyFilter(query, with deps) 
    end
    CLI->>P: BuildDownloadQueue: Verify each file → tasks for missing ones
    CLI->>DB: MarkAsUpdating (status + PID), then CLOSE database
    par download_concurrency workers
        DL->>DL: DownloadWithChecksum → temp path in pool
    end
    CLI->>DB: REOPEN database
    CLI->>P: Import each downloaded file (move=true)
    CLI->>R: FinalizeDownload: one transaction writes all Package records
    R->>DB: commit + new ref list ; Update(mirror) (Status=idle)
```

Detailed steps:

1. **Lock check.** `CheckLock()` returns *"mirror is locked by update operation, PID n"* only if
   `Status == Updating` **and** `WorkerPID` is a live process (`kill -0`). Stale locks from
   crashed runs are ignored automatically.
2. **`DownloadPackageIndexes`** builds a list of `(path, kind, component, arch)`:
   * flat: `Packages` (+ `Sources` if `-with-sources`);
   * normal: for every component × architecture `Packages` (binary), plus `debian-installer`
     Packages (udeb), plus `installer-<arch>/current/images/SHA256SUMS` (installer), plus one
     `Sources` per component.
   For each path `http.DownloadTryCompression` tries **`.bz2`, `.gz`, `.xz`, then plain**;
   a variant is only attempted if it appears in `ReleaseFiles` (unless `-ignore-checksums`);
   404/403 ⇒ try the next one. The download is checksum-verified against `ReleaseFiles` and
   transparently decompressed. Installer indexes have a special fallback: if not listed in
   `Release`, a plain `SHA256SUMS` plus detached `.gpg` is fetched and verified.
3. Stanzas are read with `ControlFileReader` → `NewPackageFromControlFile` /
   `NewUdebPackage…` / `NewSourcePackage…` / `NewInstallerPackage…` and added to
   `repo.packageList`. Duplicate name+version+arch produce a coloured *skipping package*
   warning, not an error.
4. **Filter.** If `repo.Filter != ""`: parse, `PrepareIndex()`, then `PackageList.Filter` with
   `WithDependencies = repo.FilterWithDeps`, `WithSources = repo.DownloadSources`,
   `Source = empty list` (so dependencies are searched *within the mirror only*) and the
   mirror's architectures. Output: `Packages filtered: old -> new`.
5. **`BuildDownloadQueue`**: `-latest` first reduces the list to the newest version per
   name/arch (`FilterLatest`). With `-skip-existing-packages`, packages whose key is already in
   the mirror's *old* ref list are skipped and their file info loaded from the DB (saves
   thousands of `stat` calls). Otherwise for each package `DownloadList` calls
   `packagePool.Verify(...)`; missing/bad files become tasks. Identical URLs are
   de-duplicated (extra tasks are attached as `Additional`).
6. **Mark updating and close the DB.** `MarkAsUpdating` (status + own PID) → persisted →
   `context.CloseDatabase()`. A deferred function reopens the DB and marks the mirror idle on
   *any* exit path (including ^C).
   Reason: a multi-hour download shouldn't hold the LevelDB lock; other aptly commands can run.
7. **Parallel download.** `download_concurrency` workers pull task indexes from a channel.
   Each worker: obtain temp path (`GenerateTempPath` for a local pool, else `os.CreateTemp`) →
   `DownloadWithChecksum(url, tmp, expected, ignoreChecksums)` → `task.Done = true`. Errors are
   collected (mutex), not fatal. `context.GoContextHandleSignals()` makes ^C cancel the context;
   a second ^C kills immediately.
8. **Reopen DB, import.** Sequentially `PackagePool.Import(tmp, filename, &checksums, move=true,
   checksumStorage)`; the resulting `PoolPath` and completed checksums are propagated to
   `Additional` tasks. Temp files are removed by a deferred cleanup. If interrupted or any
   download failed the command returns an error **before** finalisation — *the mirror's old
   content is untouched*, and re-running only fetches what's still missing.
9. **`FinalizeDownload`.** Opens **one DB transaction**, `UpdateInTransaction` for every package
   (writes `P`, `xF`, `xD`, `xE`), builds the new `PackageRefList` from the list, sets
   `LastDownloadDate`, commits. Then `RemoteRepoCollection.Update(repo)` stores the mirror
   record and new ref list (`E<uuid>`) and — via the deferred function — status idle.

**Atomicity property:** a mirror's visible contents (its ref list) switch from old to new in
a single batch write, and only after all files are safely in the pool. A snapshot taken at any
time sees either the complete old or the complete new state.

**Why files are verified twice:** checksum on download (integrity in flight), and `Verify`
on the next update (integrity at rest, via checksum cache).

### 5.1.4 Other mirror commands

* `mirror edit` changes `-archive-url`, `-filter` (`@file`/`@-` allowed), `-filter-with-deps`,
  `-ignore-signatures`, the `-with-sources/-udebs/-installer/-appstream` toggles and keyrings;
  `mirror rename`; `mirror drop` (refuses while snapshots were created from it unless
  `-force`); `mirror show [-with-packages] [-json]`; `mirror search`; `mirror list [-raw|-json]`.
* REST equivalents: `POST /api/mirrors`, `PUT /api/mirrors/:name` (update, asynchronous
  task), `POST /api/mirrors/:name` (edit), etc. (see [chapter 7](07-rest-api-and-tasks.md)).

---

## 5.2 Local repositories

### `aptly repo add <name> <files|dirs…>` (`cmd/repo_add.go`, `deb/import.go`)

1. Load repo (`LoadComplete`) and its package list `NewPackageListFromRefList`.
2. `deb.CollectPackageFiles` walks arguments (directories recursively, in parallel):
   `*.deb *.udeb *.ddeb *.dsc` are package candidates, `*.buildinfo` are "other files" (removed
   together with processed files but not imported).
3. `deb.ImportPackageFiles(list, files, forceReplace, verifier, pool, packageCollection,
   reporter, restriction, checksumStorageProvider)` — for each file:
   * **Read metadata**: `.dsc` → `GetControlFileFromDsc(file, verifier)` (also verifies the
     signature if present and required); `.deb`/`.udeb` → `GetControlFileFromDeb` (ar → control
     tarball → `control`).
   * Build `Package` (`NewSourcePackageFromControlFile` for `.dsc`, with `Source` renamed to
     `Package`; otherwise `NewPackageFromControlFile`/`NewUdebPackageFromControlFile`).
   * Validate name, version, architecture non-empty.
   * Compute checksums; `pool.Import(file, base, &checksums, move=false, …)`.
   * For source packages, every file listed in the `.dsc` is looked for **next to the `.dsc`**
     and imported; if absent, `pool.Verify("", name, &checksums)` looks in the pool (so
     `orig.tar.gz` can be shared between uploads).
   * Update package files/PoolPaths; apply optional `restriction` query (used by
     uploader rules); `PackageCollection.Update(p)` (transactional write).
   * `-force-replace`: search the list for same name/version/arch and remove first.
   * `list.Add(p)` — fails with `PackageConflictError` if same name/version/arch already exists
     with a different file set (reported as a failure).
   * Per-file failures are *warnings*; other files continue. `ImportPackageFiles` returns
     processed vs failed file lists.
4. `repo.UpdateRefList(NewPackageRefListFromPackageList(list))` + `LocalRepoCollection.Update`.
5. `-remove-files` deletes processed files (`.dsc`'s companions and `.buildinfo` included).
   Exit code is non-zero if any file failed.

### `aptly repo include <changes|dir…>` (`cmd/repo_include.go`, `deb/changes.go`)

Import driven by Debian `.changes` files (the output of `dpkg-buildpackage`, `dput`, …):

1. `CollectChangesFiles` finds `*.changes`.
2. `NewChanges` copies the `.changes` into a temp dir; `VerifyAndParse` verifies the PGP
   signature (unless `-ignore-signatures`; unsigned rejected unless `-accept-unsigned`) and
   parses `Files`/`Checksums-*`, `Distribution`, `Source`, `Binary`, `Architecture`.
3. `Prepare` copies every file listed in the `.changes` from the directory of the `.changes`
   file into the temp dir (files in other directories are rejected) and verifies **size, MD5,
   SHA-1 and SHA-256** against the `.changes` — any mismatch aborts that upload.
4. **Repository selection** uses the template `-repo="{{.Distribution}}"` (Go `text/template`
   over the parsed changes, e.g. `{{.Distribution}}`, `{{.Source}}`); the result names the
   local repo.
5. **Uploaders ACL** (`-uploaders-file` or the `Uploaders` stored on the repo at
   `repo create -uploaders-file`): `Uploaders.IsAllowed(changes)` walks `rules` in order. A rule
   applies if its `condition` (query language, evaluated against the `.changes` fields) matches.
   For an applicable rule: if any signature key of the `.changes` matches an entry of `deny`
   (after expanding `groups`; `"*"` matches everyone) the upload is **denied immediately**;
   else if any key matches `allow` it is **accepted**; otherwise evaluation continues with the
   next rule. If no rule accepts: *"denied as no rule matches"*.
6. Files are imported through the same `ImportPackageFiles` (with the ACL as `restriction`),
   then processed files are removed unless `-no-remove-files`.

### `repo copy / move / import / remove`

* `copy`/`move <src> <dst> <query…>`: load both lists, `Filter` the source list
  (`-with-deps` pulls dependencies not yet in the destination); add to destination (`move`
  additionally removes from source). `-dry-run` reports only.
* `import <mirror> <repo> <query…>`: same idea, but source is a **mirror** (this is how you
  cherry-pick packages from a mirror into a local repo — package files are already in the pool,
  so nothing is downloaded).
* `remove <repo> <query…>`: `list.Remove` for each match; the package **files stay in the pool**
  until `db cleanup`.
* REST: `POST /api/repos/:name/packages` and `DELETE …` take explicit package *keys*;
  `POST /api/repos/:name/copy/:src/:file` is the API for `copy`.

---

## 5.3 Snapshots

### Create (`cmd/snapshot_create.go`)

| Form | Function | Notes |
|------|----------|-------|
| `snapshot create S from mirror M` | `NewSnapshotFromRepository` | Fails with *"mirror not updated"* if the mirror has no ref list; also refuses if `CheckLock()` says an update is running. Copies `Origin/NotAutomatic/ButAutomaticUpgrades` and AppStream file map from the mirror `Meta` |
| `snapshot create S from repo R` | `NewSnapshotFromLocalRepo` | Uses the repo's ref list (empty list if none) |
| `snapshot create S empty` | `NewSnapshotFromPackageList(name, nil, empty, …)` | A blank canvas for `pull` |

Creating a snapshot writes just `S<uuid>` and `E<uuid>` — no package data moves.

### Filter (`snapshot filter <src> <dst> <query…>`)

Load package list → `PrepareIndex` → determine architectures (`-architectures` or all in the
list; required non-empty only with `-with-deps`) → `PackageList.Filter(queries ORed, WithDependencies=-with-deps,
Source=nil)` → new snapshot from the resulting list. Queries can be `@file`/`@-`
(`GetStringOrFileContent`).

### Merge (`snapshot merge <dst> <src1> <src2> …`)

```go
result := sources[0].RefList()
for i := 1..n: result = result.Merge(sources[i].RefList(), overrideMatching, false)
if latest: result.FilterLatestRefs()
```

`overrideMatching = !latest && !no-remove`. Default: **the last source wins per
(architecture, name)** — the version from a later snapshot *replaces* every version from
earlier ones. `-no-remove` keeps all versions; `-latest` keeps the highest version of each.
It never loads packages except for `-latest` comparisons (pure ref operations).

### Pull (`snapshot pull <name> <source> <dest> <query…>`)

The "controlled upgrade" primitive (`cmd/snapshot_pull.go`):

1. Load both snapshots into `PackageList`s and index them.
2. Architectures: `-architectures` or all architectures found in `<name>` (excluding source).
3. Each query is ANDed with `$Architecture ∈ archs`.
4. `sourcePackageList.Filter(queries, WithDependencies = !no-deps, Source = packageList)`:
   dependency closure is computed against *source ∪ result*, but dependencies **already
   satisfied by the destination snapshot** are not pulled again.
5. Merge into the destination list, grouped by `(arch,name)`:
   * unless `-no-remove`, all existing packages with that arch+name are **removed** first
     (the `[-] removed` lines);
   * unless `-all-matches`, only the first (= highest version, given index ordering) match per
     arch+name is added (`[+] added`).
6. `-dry-run` prints the plan; otherwise a new snapshot is created from the list with
   description *"Pulled into 'X' with 'Y' as source, pull request was: '…'"* and both snapshots
   as parents.

### Diff / Verify / Drop / Rename

* `diff A B` → `RefList.Diff` (two-pointer over refs; only loads packages for entries that differ
  to distinguish *added / removed / version-changed*). `-only-matching` limits to
  packages present in both.
* `verify S [source…]` → `PackageList.VerifyDependencies` over the snapshot (with the other
  snapshots as extra dependency *sources*) and prints unsatisfied dependencies per architecture.
* `drop S` **always** refuses if the snapshot is published (it lists the publications), and
  refuses if it is a *source of other snapshots* unless `-force`; `rename` rewrites metadata only.
* `search`/`show -with-packages` list contents; `aptly graph` renders the DAG of mirrors →
  snapshots → publications with Graphviz.

---

## 5.4 Publish

### `aptly publish snapshot|repo [flags] <name>… [[<endpoint>:]<prefix>]`
(`cmd/publish_snapshot.go` — one function serves both `snapshot` and `repo`, distinguished by
`cmd.Name()`).

1. Parse arguments: number of positional names must match the number of `-component` values
   (comma-separated); optional last argument is `[storage:]prefix` → `ParsePrefix`.
2. Load each source (`LoadComplete`); warn when a source is empty (architectures then must be
   given explicitly because they can't be inferred and can't be changed later).
3. `NewPublishedRepo(storage, prefix, distribution, architectures, components, sources, …)`:
   * source kind determined by the first source;
   * for empty `-distribution`/`-component`: `walkUpTree` climbs snapshot parents to root
     mirrors/local repos and collects their `Distribution`/`Components`/`DefaultDistribution`/
     `DefaultComponent`. If all roots agree the value is used; a single-source default component is
     `main`; a conflicting/absent distribution → *"unable to guess distribution name, please
     specify explicitly"*;
   * duplicate component names rejected; **Origin/NotAutomatic/ButAutomaticUpgrades** are
     inherited from snapshots only if they all agree;
   * prefix sanitised (`.`/`..` rejected).
4. Apply flag overrides (`-origin`, `-label`, `-suite`, `-codename`, `-notautomatic`,
   `-butautomaticupgrades`, `-skip-contents`, `-skip-bz2`, `-acquire-by-hash`, `-signed-by`,
   `-version`, `-multi-dist`) — config defaults `skip_contents_publishing` / `skip_bz2_publishing`
   apply when flags are unset.
5. `CheckDuplicate` — *(storage, prefix, distribution)* already in use ⇒ error.
6. `getSigner(flags)` builds the GPG signer (see [chapter 10](10-signing-and-verification.md));
   nil signer when `-skip-signing` or `gpg_disable_sign`.
7. **`published.Publish(...)`** — the big one, [chapter 6](06-publishing.md).
8. `PublishedRepoCollection.Add(published)` — for `local` sources also stores the ref lists
   `E<uuid><component>`.
9. Print the `deb http://your-server/<prefix> <dist> <components>` line and, for filesystem
   storages, the directory to serve.

---

## 5.5 Switch, update, drop

### `publish switch <dist> [[endpoint:]prefix] <snapshot…>` (`cmd/publish_switch.go`)

*Snapshot* publications only. Component→snapshot mapping is updated in memory
(`UpdateSnapshot` sets `rePublishing = true`), then `Publish()` regenerates everything.
Because `rePublishing` is set:

* architectures **are not recomputed** (kept from the first publication);
* index files are uploaded with a `.tmp` suffix and renamed into place at the very end
  (`indexes.RenameFiles()`), which is what makes the switch "near-atomic" for clients.

Afterwards `PublishedRepoCollection.Update` persists and — unless `-skip-cleanup` —
`CleanupPrefixComponentFiles` deletes pool files orphaned by the change.

### `publish update <dist> [[endpoint:]prefix]` (`cmd/publish_update.go`)

* **Local-repo publication:** re-reads the *current* ref list of each source local repo
  (`UpdateLocalRepo`) — publishes new state of the repo.
* **Snapshot publication:** without staged sources there's nothing to change (snapshots
  are immutable) — the command is used to apply the *staged revision* below or to re-generate
  metadata with new flags (`-skip-contents`, `-origin`, `-label`, `-multi-dist`,
  `-signed-by`, `-version`, …).

### Staged source editing: `publish source …` (`cmd/publish_source_*.go`)

`add`, `drop`, `list`, `remove`, `replace`, `update` manipulate `PublishedRepo.Revision`
(a *pending* component→source-name map) without touching the published files.
`publish update` then calls `PublishedRepo.Update` which computes
**Added / Updated / Removed** components from `Revision` vs current `Sources`
(`PublishedRepoUpdateResult`), applies them, publishes, and cleans up exactly the affected
components. `publish source drop` discards the pending revision (`DropRevision`). REST:
`/api/publish/:prefix/:distribution/sources[/:component]` and `/update`.

### `publish drop <dist> [[endpoint:]prefix]`

`PublishedRepoCollection.Remove`:

1. Determine if other publications share the same *(storage, prefix)*; if none, the whole
   `dists/` and `pool/` are removed; otherwise only `dists/<dist>` plus pool components not used by
   siblings.
2. Delete pool files of shared components that are no longer referenced
   (`CleanupPrefixComponentFiles`) unless `-skip-cleanup`.
3. Delete `U…` and (for local-repo sources) `E<uuid><component>` keys.
4. `-force-drop` continues on storage errors (leaving orphans).

---

## 5.6 DB cleanup

`aptly db cleanup` (`cmd/db_cleanup.go`, `api/db.go`) is a **mark-and-sweep garbage collector**:

```
mark 1: existing := ∪ ref lists of all mirrors, local repos, snapshots,
                      and published *local-repo* components          (Merge(…, false, true))
sweep 1: for every key in "P*" not in existing → delete P, xF, xD, xE   (one DB batch)
mark 2: referencedFiles := ⋃ pkg.FilepathList(pool) for pkg in existing  (sorted)
sweep 2: for every file in pool.FilepathList() not in referencedFiles → pool.Remove()
finish : db.CompactDB()
```

Points to know:

* It needs **exclusive access** — CLI holds the DB lock; the API takes resource `__all__` so no
  other task runs concurrently.
* `-dry-run` prints what would be deleted; `-verbose` lists objects.
* Files are removed from the **pool only**; hard-linked copies in published trees survive.
* Package rows for local-repo *publications* are included in "existing" because those refs are
  frozen copies (`E<uuid><component>`), independent of the live local repo.
* Reports total space freed (`utils.HumanBytes`); the API stores progress in a `task.Detail`
  (`TotalNumberOfPackagesToDelete`, `RemainingNumberOfPackagesToDelete`).

---

## 5.7 Other commands

| Command | What it does internally |
|---------|-------------------------|
| `aptly serve` (`cmd/serve.go`) | Requires ≥ 1 publication. Prints a suggested `deb`/`deb-src` line for every publication, then **closes the database** (`ShutdownContext`) and serves `rootDir/public` (the *default* storage only — `filesystem:` endpoints are not served) with Go's `http.FileServer` (directory listing). The docs call it "not suitable for real production usage" |
| `aptly api serve` | Runs the REST server (chapter 7) |
| `aptly graph [-format=png\|svg\|pdf…] [-layout=horizontal\|vertical] [-output=file]` | `deb.BuildGraph` creates a Graphviz digraph of mirrors, repos, snapshots, publications; renders with the external Graphviz `dot` binary; without `-output` the result is opened in a viewer |
| `aptly package search|show [-with-files -with-references]` | Queries the global `PackageCollection` (all `P…` keys); `show -with-references` lists every container referencing the package |
| `aptly db recover` | `goleveldb.RecoverDB` then checks each local repo for dangling refs (`FindDanglingReferences`) and removes them |
| `aptly config show [-yaml]` | Prints effective configuration (also converts legacy JSON → YAML) |
| `aptly task run (-filename=f \| cmd , cmd …)` | Runs several aptly commands sequentially in **one process** ("running as single thread"). Input: a file (one command per line), interactive stdin (blank line to finish), or command-line args where commands are separated by a trailing comma (`aptly task run repo create a, repo add a pkg`). For each command the DB is reopened, the command runs through the normal `Run()` path, and the context is partially cleaned up; **after the first failing command all remaining ones are skipped** and the overall result is an error |
| `aptly version` | Prints embedded version |
