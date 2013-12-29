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

Currently aptly is under heavy development, so please use it with care.

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

Configuration
-------------
