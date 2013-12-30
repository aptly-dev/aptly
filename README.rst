=====
aptly
=====

.. image:: https://travis-ci.org/smira/aptly.png?branch=master
    :target: https://travis-ci.org/smira/aptly

.. image:: https://coveralls.io/repos/smira/aptly/badge.png?branch=HEAD
    :target: https://coveralls.io/r/smira/aptly?branch=HEAD

Aptly is a swiss army knife for Debian repository management.

It allows to: ("+" means planned features)

* make mirrors of remote Debian/Ubuntu repositories, limiting by components/architectures
* take snapshots of mirrors at any point in time, fixing state of repository at some moment of time
* publish snapshot as Debian repository, ready to be consumed by apt
* merge two or more snapshots into one (+)
* filter repository by search query, pulling dependencies when required (+)
* controlled update of one or more packages in snapshot from upstream mirror, tracking dependencies (+)
* publish self-made packages as Debian repositories (+)

Current limitations:

* source packages, debian-installer and translations not supported yet
* checksums and signature are not verified while downloading
* deleting created items is not implemented
* cleaning up stale files is not implemented

Currently aptly is under heavy development, so please use it with care.

.. contents::

Download
--------

Example
-------

Usage
-----

Aptly supports commands in three basic categories:

* ``mirror``
* ``snapshot``
* ``publish``

Command ``mirror``
~~~~~~~~~~~~~~~~~~

Mirror subcommands manage mirrors of remote Debian repositories.

``aptly mirror create``
^^^^^^^^^^^^^^^^^^^^^^^

Creates mirror of remote repository. It supports only HTTP repositories.

Usage::

    $ aptly mirror create <name> <archive url> <distribution> [<component1> ...]

Params are:

* ``name`` is a name that would be used in aptly to reference this mirror
* ``archive url`` is a root of archive, e.g. http://ftp.ru.debian.org/debian/
* ``distribution`` is a distribution name, e.g. ``squeeze``
* ``component1`` is an optional list of components to download, if not 
  specified aptly would fetch all components, e.g. ``main``

Options:

* ``--architecture="i386,amd64"`` list of architectures to fetch, if not specified, 
  aptly would fetch packages for all architectures
  
Example::

  $ aptly mirror create --architecture="amd64" debian-main http://ftp.ru.debian.org/debian/ squeeze main
  2013/12/28 19:44:45 Downloading http://ftp.ru.debian.org/debian/dists/squeeze/Release...

  Mirror [debian-main]: http://ftp.ru.debian.org/debian/ squeeze successfully added.
  You can run 'aptly mirror update debian-main' to download repository contents.

``aptly mirror update``
^^^^^^^^^^^^^^^^^^^^^^^

Updates (fetches packages and meta) remote mirror. When mirror is created, it should be run for the 
first time to fetch mirror contents. This command could be run many times. If interrupted, it could
be restarted in a safe way.

Usage::

    $ aptly mirror update <name>

Params are:

* ``name`` is a mirror name (given when mirror was created)

All packages would be stored under aptly's root dir (see section on Configuration).

Example::

  $ aptly mirror update debian-main

  2013/12/29 18:32:34 Downloading http://ftp.ru.debian.org/debian/dists/squeeze/Release...
  2013/12/29 18:32:37 Downloading http://ftp.ru.debian.org/debian/dists/squeeze/main/binary-amd64/Packages.bz2...
  2013/12/29 18:37:19 Downloading http://ftp.ru.debian.org/debian/pool/main/libg/libgwenhywfar/libgwenhywfar47-dev_3.11.3-1_amd64.deb...
  ....
  
``aptly mirror list``
^^^^^^^^^^^^^^^^^^^^^

Shows list of registered mirrors of repositories.

Usage::

   $ aptly mirror list
   
Example::

   $ aptly mirror list
   List of mirrors:
    * [backports]: http://mirror.yandex.ru/backports.org/ squeeze-backports
    * [debian-main]: http://ftp.ru.debian.org/debian/ squeeze

   To get more information about repository, run `aptly mirror show <name>`.
   
``aptly mirror show``
^^^^^^^^^^^^^^^^^^^^^

Shows detailed information about mirror.

Usage::

   $ aptly mirror show <name>
   
Params are:

* ``name`` is a mirror name (given when mirror was created)

Example::

  $ aptly mirror show backports2
  Name: backports2
  Archive Root URL: http://mirror.yandex.ru/backports.org/
  Distribution: squeeze-backports
  Components: main, contrib, non-free
  Architectures: i386, amd64
  Last update: 2013-12-27 19:30:19 MSK
  Number of packages: 3898

  Information from release file:
  ...

In detailed information, one can see basiс parameters of the mirror, filters by component & architecture, timestamp
of last successful repository fetch and number of packages.

Command ``snapshot``
~~~~~~~~~~~~~~~~~~~~

Snapshot is a fixed state of remote repository. Internally snapshot is list of packages with explicit version.
Snapshot is immutable, i.e. it can't change since it has been created.

``aptly snapshot create .. from mirror``
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Creates snapshot from current state of remote mirror. Mirros should be updated at least once before using this command.

Usage::

  $ aptly snapshot create <name> from mirror <mirror-name>

Params are:

* ``name`` is a name for the snapshot to be created
* ``mirror-name`` is a mirror name (given when mirror was created)

Example::

  $ aptly snapshot create monday-updates from mirror backports2

  Snapshot monday-updates successfully created.
  You can run 'aptly publish snapshot monday-updates' to publish snapshot as Debian repository.

``aptly snapshot list``
^^^^^^^^^^^^^^^^^^^^^^^

Displays list of all created snapshots.

Usage::

  $ aptly snapshot list
  
Example::

  $ aptly snapshot list
  List of snapshots:
   * [monday-updates]: Snapshot from mirror [backports2]: http://mirror.yandex.ru/backports.org/ squeeze-backports
   * [back]: Snapshot from mirror [backports2]: http://mirror.yandex.ru/backports.org/ squeeze-backports

  To get more information about snapshot, run `aptly snapshot show <name>`.  
  
With snapshot information, basic information about snapshot origin is displayed: which mirror it has been created from.

``aptly snapshot show``
^^^^^^^^^^^^^^^^^^^^^^^

Shows detailed information about snapshot. Full list of packages in the snapshot is displayed as well.

Usage::

  $ aptly snapshot show <name>
  
Params:

* ``name`` is snapshot name which has been given during snapshot creation

Example::

  Name: back
  Created At: 2013-12-24 15:39:29 MSK
  Description: Snapshot from mirror [backports2]: http://mirror.yandex.ru/backports.org/ squeeze-backports
  Number of packages: 3898
  Packages:
    altos-1.0.3~bpo60+1_i386
    amanda-client-1:3.3.1-3~bpo60+1_amd64
    ...

Command ``publish``
~~~~~~~~~~~~~~~~~~~

Publishing snapshot as Debian repository which could be served by HTTP/FTP/rsync server. Repository is signed by
user's key with GnuPG. Key should be created beforehand (see section GPG Keys). Published repository could 
be consumed directly by apt.

``aptly publish snapshot``
^^^^^^^^^^^^^^^^^^^^^^^^^^

Usage::

  $ aptly publish snapshot <name> [<prefix>]

Params:

* ``name``
* ``prefix``

Options:

* ``-architectures=""``: list of architectures to publish (comma-separated)
* ``-component=""``: component name to publish
* ``-distribution=""``: distribution name to publish
* ``-gpg-key=""``: GPG key ID to use when signing the release

Configuration
-------------

GPG Keys
--------
