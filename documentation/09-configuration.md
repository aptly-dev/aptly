# 9. Configuration Reference

aptly is configured with a single file, **YAML (preferred since 1.6.0) or JSON** (comments and
trailing commas tolerated). The Go definition is `utils.ConfigStructure` (`utils/config.go`); each
field carries both a `json:` and a `yaml:` tag. The annotated template shipped with the package is
[`debian/aptly.conf`](../debian/aptly.conf) — it is also **embedded in the binary** and written to
`~/.aptly.conf` the first time aptly runs without any config.

Location search order (first hit wins): `-config=<file>`, `~/.aptly.conf`,
`/usr/local/etc/aptly.conf`, `/etc/aptly.conf`. Inspect the effective values with
`aptly config show` (JSON) or `aptly config show -yaml`.

> Legacy JSON keys are camelCase (`rootDir`, `downloadConcurrency`, `S3PublishEndpoints`, …); the YAML
> keys are snake_case (`root_dir`, `download_concurrency`, `s3_publish_endpoints`, …). The tables below
> list the **YAML key**, then the JSON key in parentheses.

## 9.1 General

| YAML (JSON) | Default | Meaning |
|-------------|---------|---------|
| `root_dir` (`rootDir`) | `~/.aptly` | Root for `db/`, `pool/`, `public/`, `upload/`, `skel/`. `~` is expanded |
| `log_level` (`logLevel`) | `info` | `debug`, `info`, `warning`, `error` |
| `log_format` (`logFormat`) | `default` | `default` (text) or `json`. `json` also makes the API access log JSON and disables terminal bars |
| `database_open_attempts` (`databaseOpenAttempts`) | `-1` | Retries when the DB is locked. `-1` = use the `-db-open-attempts` flag (default 10); otherwise this value overrides the flag |
| `architectures` | `[]` | Default architecture list (empty = "all available"). Overridable with `-architectures` |
| `skip_legacy_pool` (`skipLegacyPool`) | `false` (template: `true`) | OBSOLETE. Stop looking for MD5-based pre-1.1 pool paths |

## 9.2 Dependency following

Used by `snapshot pull`, `snapshot/mirror/repo … -with-deps`, `mirror` filters with deps, and the API's
`withDeps`. Each has a matching command-line flag.

| YAML (JSON) | Default | Effect |
|-------------|---------|--------|
| `dep_follow_suggests` (`dependencyFollowSuggests`) | false | Also resolve `Suggests:` |
| `dep_follow_recommends` (`dependencyFollowRecommends`) | false | Also resolve `Recommends:` |
| `dep_follow_all_variants` (`dependencyFollowAllVariants`) | false | For `a \| b` pull both alternatives (template spells it `dep_follow_allvariants`; the code reads `dep_follow_all_variants`) |
| `dep_follow_source` (`dependencyFollowSource`) | false | From a binary package also pull its source package |
| `dep_verboseresolve` (`dependencyVerboseResolve`) | false | Log resolver decisions (`Injecting package`, `Unsatisfied dependency`, …) |

> Watch the key spellings: struct tags are `dep_follow_all_variants` and `dep_verboseresolve`
> (see `utils/config.go`), whereas the commented template uses slightly different spellings. Verify
> with `aptly config show -yaml` after editing.

## 9.3 PPA short-URL expansion

| YAML | Default | Meaning |
|------|---------|---------|
| `ppa_distributor_id` | `ubuntu` | Used in `ppa:user/project` → URL |
| `ppa_codename` | *(empty → output of `lsb_release`)* | Distribution for expansion |
| `ppa_baseurl` | `http://ppa.launchpad.net` | Base URL |

## 9.4 Server

| YAML | Default | Meaning |
|------|---------|---------|
| `serve_in_api_mode` | false | API server also serves `/repos/…` from *filesystem* publish endpoints |
| `enable_metrics_endpoint` | false | Prometheus `/api/metrics` |
| `enable_swagger_endpoint` | false | Swagger UI on `/docs.html` |
| `async_api` | false | OBSOLETE. Use `?_async=true` per request |

## 9.5 Database

```yaml
database_backend:
  type: leveldb          # or: etcd
  db_path: ""            # leveldb only; "" → <root_dir>/db
  # url: "127.0.0.1:2379"  # etcd only
```

## 9.6 Mirroring

| YAML | Default | Meaning |
|------|---------|---------|
| `downloader` | `default` | `default` (net/http, streaming) or `grab` (`cavaliergopher/grab`, resumable, more robust on flaky links). Flag: `-downloader` |
| `download_concurrency` | `4` | Parallel download workers in `mirror update` |
| `download_limit` | `0` | KiB/s cap (0 = none). Flag: `-download-limit` |
| `download_retries` | `0` | Additional tries per file (`max-tries = retries + 1`). Flag: `-max-tries` |
| `download_sourcepackages` | false | Default for `-with-sources` at mirror creation |

## 9.7 Signing & verification

| YAML | Default | Meaning |
|------|---------|---------|
| `gpg_provider` | `gpg` | `gpg`/`gpg1`/`gpg2` (external binaries; see chapter 10) or `internal` (pure Go OpenPGP) |
| `gpg_disable_sign` | false | Never sign published repos |
| `gpg_disable_verify` | false | Never verify remote `Release`/`.changes` signatures |
| `gpg_keys` | `[]` | Key IDs used to sign Release when `-gpg-key` isn't provided (multiple keys ⇒ multiple signatures) |

## 9.8 Publishing defaults

| YAML | Default | Meaning |
|------|---------|---------|
| `skip_contents_publishing` | false | Default for `-skip-contents` at *first* publish |
| `skip_bz2_publishing` | false | Default for `-skip-bz2` at first publish |

(Later `publish update/switch` can change these per publication with the same flags; they are
persisted in the publication record.)

## 9.9 Published-storage endpoints

An **endpoint** is a named target; you select it on the command line/API as `type:name:prefix`
(e.g. `s3:prod:`). The default endpoint (`""`) is `<root_dir>/public` with hard links and needs no
configuration.

### filesystem

```yaml
filesystem_publish_endpoints:
  web1:
    root_dir: /srv/apt          # required
    link_method: hardlink       # hardlink | symlink | copy
    verify_method: md5          # md5 | size   (only relevant for copy)
```
`aptly publish snapshot s filesystem:web1:ubuntu`

### s3

```yaml
s3_publish_endpoints:
  prod:
    region: eu-central-1
    bucket: my-apt
    prefix: ""
    acl: public-read            # private (default) | public-read | none
    access_key_id: ""           # else AWS SDK default chain / env vars / IAM role
    secret_access_key: ""
    session_token: ""
    endpoint: ""                # S3-compatible service URL (MinIO, Ceph, …)
    storage_class: STANDARD
    encryption_method: ""       # AES256 | aws:kms
    plus_workaround: false
    disable_multidel: false
    force_sigv2: false          # accepted but currently ignored by the AWS SDK v2 based client
    force_virtualhosted_style: false
    debug: false
```

### gcs, azure, swift, jfrog

See the annotated blocks in `debian/aptly.conf`; field lists are in
[chapter 4 §4.5](04-storage.md#45-object-store-back-ends). Keys:

* `gcs_publish_endpoints.<n>`: `bucket, prefix, credentials_file, service_account_json, project, endpoint, acl, storage_class, encryption_key, disable_multidel, debug`
* `azure_publish_endpoints.<n>`: `container, prefix, account_name, account_key, endpoint`
* `swift_publish_endpoints.<n>`: `container, prefix, username, password, auth_url, tenant, tenant_id, domain, domain_id, tenant_domain, tenant_domain_id` (env fallbacks `OS_USERNAME`, `OS_PASSWORD`)
* `jfrog_publish_endpoints.<n>`: `url, repository, username, password, api_key, access_token, prefix, plus_workaround, debug` — the Artifactory repository must be of the **generic** type, not "debian".

## 9.10 Package pool storage

```yaml
packagepool_storage:
  type: local          # local (default) | azure
  path: ""             # "" → <root_dir>/pool
# --- or ---
packagepool_storage:
  type: azure
  account_name: …
  account_key: …
  container: pool1
  prefix: ""
  endpoint: ""
```

Unknown `type` values fail config loading with `unknown pool storage type`.

## 9.11 Complete example

```yaml
root_dir: /var/lib/aptly
log_level: info
architectures: [amd64, arm64]

gpg_provider: gpg
gpg_keys: ["0xABCDEF1234567890"]

downloader: grab
download_concurrency: 8
download_retries: 3

enable_metrics_endpoint: true
enable_swagger_endpoint: false

database_backend:
  type: leveldb

filesystem_publish_endpoints:
  frontend:
    root_dir: /srv/apt-public
    link_method: hardlink            # requires /srv on the same fs as /var/lib/aptly/pool,
                                     # otherwise use: copy

s3_publish_endpoints:
  cdn:
    region: us-east-1
    bucket: acme-apt
    acl: public-read
```

## 9.12 Environment variables that influence aptly

| Variable | Used by |
|----------|---------|
| `HOME` | default config location (`~/.aptly.conf`), `root_dir` default (`~/.aptly`), `~` expansion |
| `GNUPGHOME` | internal PGP provider: keyring directory (default `~/.gnupg`) |
| `SOURCE_DATE_EPOCH` | fixes the `Date:` field in published `Release` (reproducible publishes) |
| `AWS_*`, `OS_USERNAME`/`OS_PASSWORD`, `JFROG_USERNAME`/`JFROG_PASSWORD`/`JFROG_APIKEY`/`JFROG_ACCESSTOKEN` | credentials fall-backs |
| `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` | the downloader honours the standard Go proxy environment (`http.ProxyFromEnvironment`) |
| `TERM=dumb` | set by `aptly-api.service` so terminal features are avoided |
| `TMPDIR` | temp files (index generation, `.changes` handling, temporary DB for Contents) |
