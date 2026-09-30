# 12. Operations Guide

## 12.1 Directory layout on disk

```
<root_dir>/                 (default ~/.aptly; owner: the aptly user)
├── db/                     LevelDB files (CURRENT, LOCK, MANIFEST-*, *.ldb, *.log)
├── pool/                   content-addressed package files  (pool/ab/cd/<28 hex>_<file>)
├── public/                 default published storage       (hard links into pool/)
├── upload/                 API upload area                 (upload/<dir>/<files>)
└── skel/                   optional skeleton files copied into dists/<dist>/<component>/
```

`~/.aptly.conf` (or `/etc/aptly.conf`) holds configuration; GnuPG state lives in `~/.gnupg`.
Temp files use `$TMPDIR` (`/tmp`): index generation, `.changes` handling, temporary Contents DB.

## 12.2 Deployment patterns

### A. CLI on a build/release host
Simple: cron or CI job runs `aptly mirror update …; aptly snapshot create …; aptly publish switch …`.
Use `aptly task run` to run several steps in one process.

### B. API server (systemd)
Debian packaging provides `aptly-api` (`debian/aptly-api.service`, `debian/aptly-api.default`):

```
aptly-api.service   User=aptly-api Group=aptly-api  WorkingDirectory=~  LimitNOFILE=32768  Environment=TERM=dumb
/etc/default/aptly-api   LISTEN_ADDRESS='localhost:8080'   APTLY_OPTIONS="-no-lock"
ExecStart=/usr/bin/aptly api serve -config=/etc/aptly.conf ${APTLY_OPTIONS} -listen=${LISTEN_ADDRESS}
```

The post-install script creates the `aptly-api` system user (home `/var/lib/aptly-api`) and makes
`/etc/aptly.conf` `root:aptly-api 0640` because the config may contain secrets (cloud credentials,
passphrases). Note that the `aptly` package's `/etc/aptly.conf` therefore has **restrictive permissions**
(keep them so).

* The packaged default listens on **localhost** only and uses `-no-lock` (so administrators can still run
  `aptly` CLI commands while the service is idle).
* No socket unit is shipped, but `aptly api serve` adopts a single listener fd passed by systemd (you can
  write your own `aptly-api.socket` with `ListenStream=…`).

### C. Docker
`system/Dockerfile` builds a *development/test* image; for production, build a minimal image containing the
static binary, mount `root_dir` as a volume, and expose 8080 behind a reverse proxy.

### D. Serving the published repository

| Option | Notes |
|--------|-------|
| nginx/Apache over `public/` | Most common. Enable directory listing (autoindex) only if you want browsing; `Cache-Control: no-cache` (or short TTL) for `dists/`, long TTL for `pool/`. Because hard links are used, `public/` must live on the same filesystem as `pool/` or use `link_method: copy` |
| Object store (S3/GCS/Azure/Swift) | Publish directly with an endpoint; serve through the store's website endpoint or a CDN. Set `acl: public-read` or use signed access |
| `aptly serve` | Development convenience only |
| API-mode `/repos/…` | Convenience for filesystem endpoints |

Minimal nginx snippet:

```nginx
server {
  listen 80;
  root /var/lib/aptly/public;
  location / { autoindex on; }
  location ~ /dists/ { add_header Cache-Control "no-cache"; }
}
```

## 12.3 Concurrency & locking summary

| Situation | What happens |
|-----------|--------------|
| Two CLI commands at once | Second waits: retries opening LevelDB every ~10 s for `database_open_attempts` (default 10) tries, then fails *"unable to reopen the DB, maximum number of retries reached"*. Use `-db-open-attempts=0` to fail immediately (`0` loops zero times after the first try) |
| CLI while `aptly mirror update` is in its download phase | Allowed — the updater closed the DB, so other commands (listing, publishing, repo work) proceed. Commands touching *that mirror* are guarded by `CheckLock`: e.g. `snapshot create … from mirror` and another `mirror update` refuse while it is being updated |
| CLI while `aptly api serve` (default) | Blocked until the server stops |
| CLI while `aptly api serve -no-lock` | Works whenever the API is idle; API requests during a CLI command wait for the DB (they retry to open) |
| Two API mutations on the same object | Serialised by the task list (queue) |
| Two mirrors updating in parallel via API | Allowed (different resource keys) but they share the pool and network |
| `db cleanup` | Exclusive: no other API tasks run; don't run publish/update concurrently in CLI |
| Crash during mirror update | `Status=updating` with a dead PID is ignored by `CheckLock`; re-run `mirror update` (only missing files are refetched). Leftover temp files can remain in `pool/<xx>/<yy>/…` random names — `db cleanup` removes files unreferenced by any package |
| Crash during `publish` | Package files linked, indexes partially uploaded (`*.tmp` when re-publishing). Re-run the same publish command/`publish update`; stale `.tmp` files are overwritten |

## 12.4 Securing the API

The API has no built-in authentication. Recommendations:

1. Bind to `localhost` or a Unix socket (`-listen=unix:///run/aptly/aptly.sock`, restrict permissions with
   directory ownership).
2. Front with nginx/Envoy/Traefik providing **TLS + auth** (basic auth, mTLS, OIDC proxy).
3. Restrict methods/paths per role at the proxy (e.g. CI may only `POST /api/files/*`,
   `POST /api/repos/*/file/*`, `PUT /api/publish/*`).
4. Don't put passphrases in requests if the proxy logs bodies; prefer `gpg-agent` or `PassphraseFile`.
5. Limit upload size at the proxy (`client_max_body_size`); uploads are written to disk in `upload/`.
6. Run as a dedicated unprivileged user; its `~/.gnupg` holds the signing key.

## 12.5 Backup, restore, migration

* **What to back up:** `root_dir/db` (metadata), `root_dir/pool` (package files), GnuPG keys, config.
  `public/` is *derived* (re-publishable) — but the exact bytes of indexes matter to clients that pinned
  hashes, so many operators back it up too.
* **Consistent backup:** stop aptly (or make sure no CLI/API task runs); LevelDB files are only
  consistent when the DB is closed. Copy `db/` first, then `pool/` (extra files in pool are harmless; missing files are not).
* **Restore:** put back both; run `aptly db cleanup -dry-run` to check consistency; `aptly db recover` if
  the DB reports corruption (rebuilds LevelDB and drops dangling refs from local repos).
* **Moving to another host:** copy `db/` and `pool/`; publishes with hard links need to be regenerated
  (`aptly publish update`/`switch`) since `public/` links won't survive a cross-fs copy.
* **LevelDB → etcd:** `system/leveldb2etcd.py` helper.
* **Reproducibility:** *snapshots are the backup of intent*. Keep snapshot names and publication commands in
  version control.

## 12.6 Disk & performance

* Disk = pool (dominant) + DB + (public if using `copy` links). Mirrors of full archives are tens to
  hundreds of GB per suite/arch; use `-filter`, `-architectures`, skip `-with-sources`, and `mirror update -latest`.
* `download_concurrency` 4–16 for fast links; `downloader: grab` for flaky networks; `download_limit` to be
  polite; `-skip-existing-packages` to speed up repeated updates of big mirrors.
* Publishing: `-skip-contents` and `-skip-bz2` drastically reduce time on large repos.
* Many publications sharing one prefix slow down `switch`/`update` (cleanup scans siblings) — prefer distinct
  prefixes or `-multi-dist` for big estates.
* LevelDB: `db cleanup` compacts. `aptly db cleanup` on multi-million-package DBs can take minutes;
  schedule off-peak.
* Memory: package lists hold all packages of the sources in RAM, so large merges/publishes need
  proportionally more memory; a build with `EnableDebug` adds `-memprofile`/`-memstats` flags for
  investigation.
* Put `root_dir` on SSD: LevelDB & index generation are IO-bound.

## 12.7 Monitoring

* `GET /api/ready`, `GET /api/healthy` (liveness/readiness probes).
* `GET /api/storage` (disk usage of root_dir filesystem) — alert at e.g. 85 %.
* Prometheus: enable `enable_metrics_endpoint`; alert on `aptly_api_http_requests_total{code=~"5.."}` and on
  `aptly_repos_package_count` drops.
* Logs: `log_format: json` for machine parsing; task output via `/api/tasks/:id/output`.

## 12.8 Routine maintenance checklist

| Frequency | Task |
|-----------|------|
| Each mirror refresh | `mirror update` → `snapshot create` → `snapshot verify` → publish/switch |
| Weekly | `aptly db cleanup` after deleting old snapshots; check `/api/storage` |
| On key rotation | Publish with two `-gpg-key`s, update clients' keyrings, then drop the old key |
| Before upgrades | Back up `db/`; upgrade aptly (DB formats are auto-migrated on read, see §3.3) |
| Quarterly | Test restore procedure; prune obsolete snapshots/publications |

## 12.9 Troubleshooting

| Symptom | Likely cause / fix |
|---------|-------------------|
| `unable to reopen the DB, maximum number of retries reached` | Another aptly process (or `api serve` without `-no-lock`) holds the DB. Stop it, or use `-no-lock` / etcd |
| `mirror is locked by update operation, PID N` | Update in progress; if stale (dead PID) it is ignored automatically; use `-force` otherwise |
| `Looks like your keyring with trusted keys is empty` / `Release signature verification failed` | Import archive keys into **`trustedkeys.gpg`** (see chapter 10) or use `-keyring` |
| `unable to figure out list of architectures, please supply explicit list` | Publishing from an empty source: add `-architectures=amd64` |
| `error linking file to …: file already exists and is different` | Different bytes with same pool path in a shared prefix. Use `-multi-dist`, distinct prefixes, or (carefully) `-force-overwrite` |
| `cannot link … from non-local pool` | Azure pool + hardlink/symlink publish endpoint ⇒ set `link_method: copy` |
| `Hash Sum mismatch` on clients after re-publish | Client fetched during the rename window: enable `-acquire-by-hash`; or retry |
| `file … already exists` on `Import` with different size | Corrupt/partial file in the pool; delete the pool file and re-run the update |
| `prefix/distribution already used by another published repo` | Use `publish switch/update` or choose another prefix |
| `unable to publish: … no such file` in pool | Pool file removed manually; `mirror update` re-downloads missing files (it verifies), `repo` packages must be re-added |
| Publishing prompts for a passphrase / hangs (API) | API is batch-only; use gpg-agent or `Signing.PassphraseFile` |
| Huge memory in `snapshot merge/pull` | Load of all packages; use `-latest`/filters, more RAM |
| `aptly` says `Config file not found, creating default config…` | Expected the first run; edit `~/.aptly.conf` |
| Old `.deb` files linger in `public/pool` | `publish update` w/o `-skip-cleanup` removes orphans; drop old publication; hard-linked files persist after `db cleanup` until publications are cleaned |
| `Filter … unsatisfied dependency` (verbose) | Enable `-dep-verbose-resolve`; add source snapshot providing it |

Add `log_level: debug` (or `-dep-verbose-resolve`) for more diagnostics; S3/GCS/JFrog endpoints have
`debug: true` to dump requests.
