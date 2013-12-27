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

Example
-------

Usage
-----

Aptly supports commands in three basic categories:

* `mirror`
* `snapshot`
* `publish`

Command `mirror`
~~~~~~~~~~~~~~~~

Mirror subcommands manage mirrors of remote Debian repositories.

`aptly mirror create`
^^^^^^^^^^^^^^^^^^^^^

Creates mirror of remote repository. It supports only HTTP repositories.

Usage::

    $ aptly mirror create <name> <archive url> <distribution> [<component1> ...]
