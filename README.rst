=====
aptly
=====

.. image:: https://travis-ci.org/smira/aptly.png?branch=master
    :target: https://travis-ci.org/smira/aptly

.. image:: https://coveralls.io/repos/smira/aptly/badge.png?branch=HEAD
    :target: https://coveralls.io/r/smira/aptly?branch=HEAD

Aptly is a swiss army knife for Debian repository management.

Documentation is available at `http://www.aptly.info/ <http://www.aptly.info/>`_. For support use
mailing list `aptly-discuss <https://groups.google.com/forum/#!forum/aptly-discuss>`_.

Aptly features: ("+" means planned features)

* make mirrors of remote Debian/Ubuntu repositories, limiting by components/architectures
* take snapshots of mirrors at any point in time, fixing state of repository at some moment of time
* publish snapshot as Debian repository, ready to be consumed by apt
* controlled update of one or more packages in snapshot from upstream mirror, tracking dependencies
* merge two or more snapshots into one (+)
* filter repository by search query, pulling dependencies when required (+)
* publish self-made packages as Debian repositories (+)

Current limitations:

* source packages, debian-installer and translations not supported yet
* checksums and signature are not verified while downloading
* deleting created items is not implemented
* cleaning up stale files is not implemented

Currently aptly is under heavy development, so please use it with care.

Download
--------

Binary executables (depends almost only on libc) are available for download from `Bintray <https://bintray.com/smira/generic/aptly>`_.

If you have Go environment set up, you can build aptly from source by running::

    go get -d -v github.com/smira/aptly
    go install github.com/smira/aptly