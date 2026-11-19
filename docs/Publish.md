# Publish Repositories, Snapshots, Mirrors
<div>

Publish snapshot or local repo as Debian repository to be used as APT source on Debian based systems.

The published repository is signed with the user's GnuPG key.

Repositories can be published to local directories, Amazon S3 buckets, Azure, Swift, or JFrog Artifactory Storage.

#### GPG Keys

GPG key is required to sign any published repository. The key pari should be generated before publishing.

Public part of the key should be exported from your keyring using `gpg --export --armor` and imported on the system which uses a published repository.

* Multiple signing keys can be defined in aptly.conf using the gpgKeys array:
```
"gpgKeys": [
    "KEY_ID_x",
    "KEY_ID_y"
]
```

* It is also possible to pass multiple keys via the CLI using the repeatable `--gpg-key` flag:
```
aptly publish repo my-repo --gpg-key=KEY_ID_a --gpg-key=KEY_ID_b
```
* When using the REST API, the `gpgKey` parameter supports a comma-separated list of key IDs:
```
"gpgKey": "KEY_ID_a,KEY_ID_b"
```
* If `--gpg-key` is specified on the command line, or `gpgKey` is provided via the REST API, it takes precedence over any gpgKeys configuration in aptly.conf.
* With multi-key support, aptly will sign all Release files (both clearsigned and detached signatures) with each provided key, ensuring a smooth key rotation process while maintaining compatibility for existing clients.

#### Parameters

Publish APIs use following convention to identify published repositories: `/api/publish/:prefix/:distribution`.  `:distribution` is distribution name, while `:prefix` is `[<storage>:]<prefix>` (storage is optional, it defaults to empty string), if publishing prefix contains slashes `/`, they should be replaced with underscores (`_`) and underscores
should be replaced with double underscore (`__`). To specify root `:prefix`, use `:.`, as `.` is ambiguous in URLs.

#### Performance-related environment variables

The following environment variables tune publish and drop performance. They are intended for installations where aptly publishes to a local filesystem and can aggressively use local CPU, memory, and filesystem throughput.

| Variable | Default | Description |
| --- | --- | --- |
| `APTLY_PARALLEL_COMPONENTS` | enabled | Controls concurrent processing of publish components, for example `main`, `restricted`, `universe`, and `multiverse`. Set to `0` to disable component-level parallelism. |
| `APTLY_PARALLEL_PACKAGES` | enabled | Controls concurrent package processing inside each component. Set to `0` to disable package-level parallelism. Package index output order is preserved. |
| `APTLY_PARALLEL_PACKAGE_WORKERS` | number of CPU cores | Overrides the number of package workers used when `APTLY_PARALLEL_PACKAGES` is enabled. For example, set `APTLY_PARALLEL_PACKAGE_WORKERS=4` to force four package workers. |
| `APTLY_FAST_REMOVE` | enabled | Uses external `rm -rf -- <path>` instead of Go `os.RemoveAll` for local filesystem cleanup. Set to `0` to restore Go's built-in removal path. |
| `APTLY_ASYNC_REMOVE` | enabled | Makes local filesystem removals asynchronous. Removed paths are first renamed into `.aptly-trash/<batch-id>/`; the aptly DB is updated; physical deletion starts only after the DB update succeeds. Set to `0` to make cleanup synchronous. |

`APTLY_ASYNC_REMOVE` changes the operational semantics of `publish drop`: when the command returns, the published repository has already been removed from aptly metadata and moved out of the served path, but disk space may still be reclaimed in the background by `rm -rf`. If the DB update fails, staged removals are rolled back to their original paths.

Examples:

```
# Disable asynchronous removal and wait for physical cleanup before returning.
APTLY_ASYNC_REMOVE=0 aptly publish drop noble filesystem:public:test-ubuntu

# Disable publish parallelism for troubleshooting.
APTLY_PARALLEL_COMPONENTS=0 APTLY_PARALLEL_PACKAGES=0 aptly publish snapshot ...

# Override package worker count.
APTLY_PARALLEL_PACKAGE_WORKERS=4 aptly publish snapshot ...

# Use Go os.RemoveAll instead of external rm -rf.
APTLY_ASYNC_REMOVE=0 APTLY_FAST_REMOVE=0 aptly publish drop noble filesystem:public:test-ubuntu
```

</div>
