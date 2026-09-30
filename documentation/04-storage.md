# 4. Package Pool & Published Storage

aptly separates two storage concepts that are often conflated:

| | **Package pool** | **Published storage** |
|-|------------------|-----------------------|
| Purpose | Authoritative, de-duplicated store of *every package file aptly knows about* | The tree that apt clients actually download (`dists/`, `pool/`) |
| Interface | `aptly.PackagePool` (+ optional `LocalPackagePool`) | `aptly.PublishedStorage` |
| Count | Exactly **one** per aptly instance | **Any number** (named endpoints) |
| Addressing | Content-addressed (SHA-256) | Human/apt layout (`pool/<component>/<x>/<source>/file.deb`) |
| Back-ends | Local filesystem, Azure Blob | Filesystem, S3, GCS, Azure Blob, Swift, JFrog Artifactory |
| Written by | mirror update, `repo add/include` | `publish snapshot/repo/update/switch` |
| Removed by | `db cleanup` | `publish drop`, `publish update/switch` cleanup |

## 4.1 The local package pool (`files/package_pool.go`)

### Layout

For a file with SHA-256 `476e0cdac6bc757dd2b78bacc1325323b09c45ecb41d4562deec2a1c7c148405`
and name `my-package_1.2.3_all.deb`:

```
<pool>/47/6e/0cdac6bc757dd2b78bacc13253_my-package_1.2.3_all.deb
        │  │   └── sha256[4:32] + "_" + basename
        │  └────── sha256[2:4]
        └───────── sha256[0:2]
```

(`buildPoolPath`: `hash[0:2]/hash[2:4]/hash[4:32] + "_" + filename`.) The pool root is
`rootDir/pool` unless `packagepool_storage.path` says otherwise. See also
[`files/README.md`](../files/README.md).

**Legacy layout** (aptly ≤ 1.0): `<md5[0:2]>/<md5[2:4]>/<filename>`. It is still *read* unless
`skip_legacy_pool: true` (the default for fresh configs); new files are never written that
way. Package records keep the `PoolPath` they were imported with, so mixed pools work.

### Operations

| Method | Behaviour |
|--------|-----------|
| `Import(src, basename, checksums, move, checksumStorage)` | Serialised by a mutex. Computes any missing checksums (MD5+SHA-256 mandatory; recomputes if size mismatched). If the target exists **with identical size**, it is trusted after `ensureChecksums`, and its path is returned (de-duplication); if size differs → error *"file … already exists"*. If a legacy path exists with same size, that path is returned. Otherwise it creates the directories and **tries `os.Link` (hard link) when source and pool are on the same device**, falling back to a copy. Finally it records checksums in the checksum cache and, if `move`, removes the source |
| `Verify(poolPath, basename, checksums, checksumStorage)` | Existence + size + checksum check. With no `poolPath` it *guesses* candidates from SHA-256 (modern) and MD5 (legacy). Fills back the full `ChecksumInfo` into the argument on success. Returns `(path, found, err)` |
| `Open`, `Size`, `Stat` | Direct file access |
| `Remove(path)` | Deletes the file, returns its size |
| `FilepathList(progress)` | Parallel walk (`saracen/walker`) of the whole pool for GC; returns sorted relative paths |
| `GenerateTempPath(name)` | Returns a random path *inside the pool root* — `<pool>/<uuid[0:2]>/<uuid[2:4]>/<uuid[4:]><filename>` — so a downloaded file sits on the same filesystem as its final location and `Import` can **hard-link** it instead of copying |
| `Link`, `Symlink`, `FullPath` | Used by `files.PublishedStorage.LinkFromPool` for hardlink/symlink publishing |

### Checksum cache

Verifying a multi-terabyte pool by re-hashing would be prohibitive. `ensureChecksums` looks
up `C<poolPath>` in the DB (`ChecksumCollection`) and only hashes the file when absent, then
stores `ChecksumInfo{Size, MD5, SHA1, SHA256, SHA512}`. Consequence: **if you replace a pool
file behind aptly's back with different bytes of the same size, aptly will not notice** until
the cache entry is dropped.

### What "mirror update" and "repo add" do with the pool

* **Mirror update** first asks `Verify` for each file of each package; only missing/mismatching
  files are queued for download (`Package.DownloadList`). Downloads land in temp paths from
  `GenerateTempPath`, are checksum-verified, then `Import`ed with `move=true`.
* **repo add/include/`POST /api/repos/:name/file/...`** (`deb.ImportPackageFiles`) call `Import`
  with the user's file as the source and `move=false`; removal of successfully imported source
  files is a separate step: CLI `-remove-files` (add) / `-no-remove-files` (include); REST removes them
  after import unless `noRemove=1` / `noRemoveFiles=1`. The file appears in the pool exactly once regardless of how many repos add it.

## 4.2 The Azure package pool (`azure/package_pool.go`)

Selecting

```yaml
packagepool_storage:
  type: azure
  account_name: …
  account_key: …
  container: pool
  prefix: ""
  endpoint: ""
```

stores pool files as blobs with **the same relative path scheme** as the local pool. Differences
from the local pool:

* No `LocalPackagePool` methods → files cannot be hard-linked or symlinked; publishing to a
  filesystem endpoint must use `link_method: copy`, and the cloud back-ends stream the blob
  through `Open`.
* No legacy (MD5) layout (`LegacyPath` returns an empty path).
* `Import` uploads (the `move` flag is ignored); `GenerateTempPath` isn't available so the
  mirror downloader uses `os.CreateTemp`.
* `Remove`/`FilepathList` use blob APIs.

## 4.3 Published storage interface

```go
MkDir(path)                       RemoveDirs(path, progress)          Remove(path)
PutFile(path, sourceFilename)     RenameFile(old, new)                Filelist(prefix)
LinkFromPool(prefix, relPath, fileName, sourcePool, sourcePath, checksums, force)
SymLink(src, dst)  HardLink(src, dst)  FileExists(path)  ReadLink(path)
```

* Paths are relative to the storage root (plus the configured `prefix` for cloud back-ends).
* `LinkFromPool` is the *only* way package files reach the published tree; it is responsible
  for idempotency: if the destination already exists it must decide **same / different**
  (inode for hard links, MD5 or size for copies, MD5 metadata for object stores). Different
  content is an error unless `force` (`-force-overwrite`).
* `RenameFile` provides the "atomic-ish" swap of `*.tmp` index files during re-publish.
  Object stores implement it as copy + delete.
* `SymLink/HardLink/ReadLink` exist for `by-hash` and multi-distribution helpers; object
  stores emulate them (copy + metadata / text marker).

### Naming published storage

The first argument (`[storage:]prefix`) of publish commands is parsed by `deb.ParsePrefix`:

| Value | Storage | Prefix |
|-------|---------|--------|
| `debian` | default (`""` → `rootDir/public`, hardlink) | `debian` |
| `.` | default | `.` (root) |
| `filesystem:mirror1:ubuntu` | `FileSystemPublishRoots["mirror1"]` | `ubuntu` |
| `s3:prod:` | `S3PublishRoots["prod"]` | `.` |
| `gcs:x:…`, `swift:x:…`, `azure:x:…`, `jfrog:x:…` | respective map | |

Parsing splits at the **last** colon; an empty prefix after a colon means `.`. Leading/trailing
`/` are trimmed. `NewPublishedRepo` additionally rejects prefixes containing `.`/`..` path
components. `context.GetPublishedStorage(name)` instantiates each named storage once per
process and caches it.

## 4.4 Filesystem published storage (`files/public.go`)

Config (`filesystem_publish_endpoints.<name>`): `root_dir`, `link_method`
(`hardlink` default | `symlink` | `copy`), `verify_method` (`md5` default | `size`, only for
`copy`). The implicit default storage is `rootDir/public` with `hardlink`.

`LinkFromPool` logic:

1. Compute `destination = root/prefix/relPath/fileName` and `mkdir -p` its directory.
2. Non-copy methods require the source pool to be a `LocalPackagePool`.
3. If destination exists:
   * `copy`: compare source size to destination size (`size` mode) or compare
     destination MD5 to the source MD5 (`md5` mode); equal ⇒ nothing to do;
   * link/symlink: equal **inode and device** ⇒ nothing to do (a symlink to a different
     filesystem could reuse inode numbers, hence the device check);
   * otherwise error, unless `force`, in which case the destination is removed first.
4. Create: for `copy` stream from `sourcePool.Open` then **`fsync`** (to surface `ENOSPC`);
   `symlink` → `pool.Symlink`; default → `pool.Link` (hard link).

Hard links are the default and make a publication essentially free in disk space and instant,
**but** they require the pool and the publish root to be on the same filesystem, and a hard
link keeps the data alive even after `db cleanup` removes it from the pool — which is
actually what protects still-served files.

`RemoveDirs` uses `os.RemoveAll`; `Filelist` walks in parallel and returns sorted relative
file names.

## 4.5 Object-store back-ends

All object stores share a pattern:

* `MkDir` is a no-op (flat namespaces).
* Uploads are keyed `prefix + path`.
* `LinkFromPool` = *stream from `sourcePool.Open()` to `PutObject`* with skip-if-same logic
  keyed on **MD5**, because MD5 is available as ETag or custom metadata.
* Directory removal = list by prefix + batch delete.

| Back-end | Package | Config section | Specifics |
|----------|---------|----------------|-----------|
| **Amazon S3 / S3-compatible** | `s3` (AWS SDK v2) | `s3_publish_endpoints` | `bucket`, `region`, `endpoint` (MinIO etc.), `prefix`, `acl` (default `private`; use `public-read` for direct serving; `none` to omit), `storage_class`, `encryption_method` (`AES256`/`aws:kms`), `plus_workaround`, `disable_multidel`, `force_virtualhosted_style`, `debug`; credentials from config or the SDK's default chain (env, IAM role…). **Path cache**: on first `LinkFromPool` under a prefix it lists `prefix/pool` once and caches path → MD5 to avoid a HEAD per file. If the ETag is not a plain MD5 (multipart or SSE-KMS) it falls back to a `HEAD` reading the `Md5` user-metadata that aptly stored. `plus_workaround` writes a second copy of any key containing `+` with `+`→space, because some S3 front-ends decode `+` in URLs as a space. `RenameFile` = `CopyObject` + `DeleteObject`. `SymLink` = copy with metadata |
| **Google Cloud Storage** | `gcs` | `gcs_publish_endpoints` | `bucket`, `prefix`, `credentials_file` or inline `service_account_json`, `project`, `endpoint`, `acl`, `storage_class`, `encryption_key`, `disable_multidel`. Same MD5-based skip logic |
| **Azure Blob** | `azure` | `azure_publish_endpoints` | `account_name`, `account_key`, `container`, `prefix`, `endpoint`; `HardLink` implemented as symlink emulation; `SymLink` writes link metadata; per-prefix path→MD5 cache |
| **OpenStack Swift** | `swift` | `swift_publish_endpoints` | Keystone authentication (`username`, `password`, `auth_url`, `tenant`, `tenant_id`, `domain`, `domain_id`, `tenant_domain`, `tenant_domain_id`), `container`, `prefix`. Uses the object `Hash` (MD5) to compare |
| **JFrog Artifactory** | `jfrog` | `jfrog_publish_endpoints` | `repository`, `url`, one of `user`+`password` / `api_key` / `access_token` (also from env `JFROG_USERNAME`, `JFROG_PASSWORD`, `JFROG_APIKEY`, `JFROG_ACCESSTOKEN`), `prefix`, `plus_workaround`, `debug`. Materialises non-local pool sources in a temp file before upload |

Because object stores are "serve by URL", the resulting bucket/container is what you point
`apt` at (with public read or a CDN/proxy).

## 4.6 Where files end up in a published tree

```
<storage>/<prefix>/
├── dists/<distribution>/
│   ├── Release            Release.gpg          InRelease
│   ├── Contents-<arch>.gz (legacy, top-level)
│   └── <component>/
│       ├── binary-<arch>/{Packages,Packages.gz,Packages.bz2,Release}
│       ├── debian-installer/binary-<arch>/…      (udebs)
│       ├── source/{Sources,Sources.gz,Sources.bz2,Release}
│       ├── Contents-<arch>.gz
│       ├── installer-<arch>/current/{images|legacy-images}/…
│       └── (skeleton files / AppStream dep11 files)
└── pool/<component>/<x>/<source>/<file>          (multi-dist: pool/<distribution>/<component>/…)
```

`<x>` is the first letter of the source package name, or the first four characters for
`lib*` sources (`Package.PoolDirectory`). If `Source` carries a version (`foo (1.2)`), only the
name part is used. `MultiDist` publications place files under `pool/<distribution>/…` so the
same *filename* with different contents can coexist in different distributions of one prefix.
Details of generation are in [chapter 6](06-publishing.md).

## 4.7 Cleaning up

* **Published storage**: after `publish update/switch`, `CleanupPrefixComponentFiles` deletes
  pool files under `prefix/pool/<component>/` that are no longer referenced by *any*
  publication sharing the same storage+prefix (skippable with `-skip-cleanup`). `publish drop`
  removes `dists/<dist>` and unreferenced pool files (whole prefix removed if it was the last
  publication).
* **Pool**: `aptly db cleanup` (mark-and-sweep, [chapter 5](05-workflows.md#56-db-cleanup)).

Both sweeps are conservative: they compute "referenced" sets from the DB rather than
inspecting file contents.
