# Repository and Mirror Snapshots
<div>
Snapshot is a fixed state of remote repository mirror or local repository. Internally snapshot is list of references to packages.
Snapshot is immutable, i.e. it can't be changed since it has been created. Snapshots could be [merged](/doc/aptly/snapshot/merge/),
[filtered](/doc/aptly/snapshot/pull/),
individual packages could be [pulled](/doc/aptly/snapshot/pull/), snapshot could be
[verified](/doc/aptly/snapshot/verify/) for missing dependencies. Finally, snapshots could be
[published as repositories](/doc/aptly/publish/snapshot)
</div>
