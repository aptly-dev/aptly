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
* merge two or more snapshots into one
* filter repository by search query, pulling dependencies when required (+)
* publish self-made packages as Debian repositories (+)
* mirror repositories "as-is" (without resigning with user's key) (+)
* support for yum repositories (+)

Current limitations:

* debian-installer and translations not supported yet

Currently aptly is under heavy development, so please use it with care.

Download
--------

Binary executables (depends almost only on libc) are available for download from `Bintray <http://dl.bintray.com/smira/aptly/>`_.

If you have Go environment set up, you can build aptly from source by running (go 1.1+ required)::

    go get github.com/smira/aptly

If you don't have Go installed (or older version), you can easily install Go using `gvm <https://github.com/moovweb/gvm/>`_.
