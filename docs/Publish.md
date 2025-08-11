# Publish Repositories, Snapshots, Mirrors
<div>

Publish snapshot or local repo as Debian repository to be used as APT source on Debian based systems.

The published repository is signed with the user's GnuPG key.

Repositories can be published to local directories, Amazon S3 buckets, Azure or Swift Storage.

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
* If `--gpg-key` is specified on the command line, it takes precedence over any gpgKeys configuration in `aptly.conf`.
* With multi-key support, aptly will sign all Release files (both clearsigned and detached signatures) with each provided key, ensuring a smooth key rotation process while maintaining compatibility for existing clients.

#### Parameters

Publish APIs use following convention to identify published repositories: `/api/publish/:prefix/:distribution`.  `:distribution` is distribution name, while `:prefix` is `[<storage>:]<prefix>` (storage is optional, it defaults to empty string), if publishing prefix contains slashes `/`, they should be replaced with underscores (`_`) and underscores
should be replaced with double underscore (`__`). To specify root `:prefix`, use `:.`, as `.` is ambigious in URLs.

</div>
