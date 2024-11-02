# Aptly Publish Points
<div>

Publish snapshot or local repo as Debian repository which could be served by HTTP/FTP/rsync server. Repository is signed by user's key with GnuPG. Key should be created beforehand (see section GPG Keys below).  Published repository could be consumed directly by apt.

Repositories could be published to Amazon S3 service: create bucket,
[configure publishing endpoint](/doc/feature/s3/) and use S3 endpoint when
publishing.


#### GPG Keys

GPG key is required to sign any published repository. Key should be generated before publishing first repository.

Key generation, storage, backup and revocation is out of scope of this document, there are many tutorials available, e.g. [this one](http://fedoraproject.org/wiki/Creating_GPG_Keys).

Publiс part of the key should be exported from your keyring using `gpg --export --armor` and imported into apt keyring using `apt-key` tool on all machines that would be using published repositories.

Signing releases is highly recommended, but if you want to skip it, you can either use `gpgDisableSign` configuration option or `--skip-signing` flag.

#### Parameters

Publish APIs use following convention to identify published repositories: `/api/publish/:prefix/:distribution`.  `:distribution` is distribution name, while `:prefix` is `[<storage>:]<prefix>` (storage is optional, it defaults to empty string), if publishing prefix contains slashes `/`, they should be replaced with underscores (`_`) and underscores
should be replaced with double underscore (`__`). To specify root `:prefix`, use `:.`, as `.` is ambigious in URLs.

</div>
