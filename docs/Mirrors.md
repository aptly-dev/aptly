# Repository Mirrors
<div>
aptly allows to create mirrors of remote Debian repositories, HTTP, HTTPS and FTP repositories are supported.

Mirrors are created with [`aptly mirror create`](/doc/aptly/mirror/create/) command, mirror contents are downloaded with [`aptly mirror update`](/doc/aptly/mirror/update/) command. Mirror could be updated at any moment. In order to preserve current mirror state, [create snapshot](/doc/aptly/snapshot/create/) of the mirror. Snapshot could be published or used to build other snapshots.
</div>

