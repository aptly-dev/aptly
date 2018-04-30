=====
aptly
=====

.. image:: https://api.travis-ci.org/aptly-dev/aptly.svg?branch=master
    :target: https://travis-ci.org/aptly-dev/aptly

.. image:: https://codecov.io/gh/aptly-dev/aptly/branch/master/graph/badge.svg
  :target: https://codecov.io/gh/aptly-dev/aptly

.. image:: https://badges.gitter.im/Join Chat.svg
    :target: https://gitter.im/aptly-dev/aptly?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge

.. image:: http://goreportcard.com/badge/aptly-dev/aptly
    :target: http://goreportcard.com/report/aptly-dev/aptly

Aptly is a swiss army knife for Debian repository management.

.. image:: http://www.aptly.info/img/aptly_logo.png
    :target: http://www.aptly.info/

Documentation is available at `http://www.aptly.info/ <http://www.aptly.info/>`_. For support please use
mailing list `aptly-discuss <https://groups.google.com/forum/#!forum/aptly-discuss>`_.

Aptly features: ("+" means planned features)

* make mirrors of remote Debian/Ubuntu repositories, limiting by components/architectures
* take snapshots of mirrors at any point in time, fixing state of repository at some moment of time
* publish snapshot as Debian repository, ready to be consumed by apt
* controlled update of one or more packages in snapshot from upstream mirror, tracking dependencies
* merge two or more snapshots into one
* filter repository by search query, pulling dependencies when required
* publish self-made packages as Debian repositories
* REST API for remote access
* mirror repositories "as-is" (without resigning with user's key) (+)
* support for yum repositories (+)

Current limitations:

* translations are not supported yet

Download
--------

To install aptly on Debian/Ubuntu, add new repository to ``/etc/apt/sources.list``::

    deb http://repo.aptly.info/ squeeze main

And import key that is used to sign the release::

    $ apt-key adv --keyserver pool.sks-keyservers.net --recv-keys ED75B5A4483DA07C

After that you can install aptly as any other software package::

    $ apt-get update
    $ apt-get install aptly

Don't worry about squeeze part in repo name: aptly package should work on Debian squeeze+,
Ubuntu 10.0+. Package contains aptly binary, man page and bash completion.

If you would like to use nightly builds (unstable), please use following repository::

    deb http://repo.aptly.info/ nightly main

Binary executables (depends almost only on libc) are available for download from `Bintray <http://dl.bintray.com/smira/aptly/>`_.

If you have Go environment set up, you can build aptly from source by running (go 1.8+ required)::

    mkdir -p $GOPATH/src/github.com/aptly-dev/aptly
    git clone https://github.com/aptly-dev/aptly $GOPATH/src/github.com/aptly-dev/aptly
    cd $GOPATH/src/github.com/aptly-dev/aptly
    make install

Binary would be installed to ```$GOPATH/bin/aptly``.

Contributing
------------

Please follow detailed documentation in `CONTRIBUTING.md <CONTRIBUTING.md>`_.

Integrations
------------

Vagrant:

-   `Vagrant configuration <https://github.com/sepulworld/aptly-vagrant>`_ by
    Zane Williamson, allowing to bring two virtual servers, one with aptly installed
    and another one set up to install packages from repository published by aptly

Docker:

-    `Docker container <https://github.com/mikepurvis/aptly-docker>`_ with aptly inside by Mike Purvis
-    `Docker container <https://github.com/bryanhong/docker-aptly>`_ with aptly and nginx by Bryan Hong

With configuration management systems:

-   `Chef cookbook <https://github.com/hw-cookbooks/aptly>`_ by Aaron Baer
    (Heavy Water Operations, LLC)
-   `Puppet module <https://github.com/alphagov/puppet-aptly>`_ by
    Government Digital Services
-   `Puppet module <https://github.com/tubemogul/puppet-aptly>`_ by
    TubeMogul
-   `SaltStack Formula <https://github.com/saltstack-formulas/aptly-formula>`_ by
    Forrest Alvarez and Brian Jackson
-   `Ansible role <https://github.com/aioue/ansible-role-aptly>`_ by Tom Paine

CLI for aptly API:

-   `Ruby aptly CLI/library <https://github.com/sepulworld/aptly_cli>`_ by Zane Williamson
-   `Python aptly CLI (good for CI) <https://github.com/TimSusa/aptly_api_cli>`_ by Tim Susa

GUI for aptly API:

-   `Python aptly GUI (via pyqt5) <https://github.com/chnyda/python-aptly-gui>`_ by Cedric Hnyda

Scala sbt:

-   `sbt aptly plugin <https://github.com/amalakar/sbt-aptly>`_ by Arup Malakar
