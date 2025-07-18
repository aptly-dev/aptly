.. image:: https://github.com/aptly-dev/aptly/actions/workflows/ci.yml/badge.svg
    :target: https://github.com/aptly-dev/aptly/actions

.. image:: https://codecov.io/gh/aptly-dev/aptly/branch/master/graph/badge.svg
    :target: https://codecov.io/gh/aptly-dev/aptly

.. image:: https://badges.gitter.im/Join Chat.svg
    :target: https://matrix.to/#/#aptly:gitter.im

.. image:: https://goreportcard.com/badge/github.com/aptly-dev/aptly
    :target: https://goreportcard.com/report/aptly-dev/aptly

aptly
=====

Aptly is a swiss army knife for Debian repository management.

.. image:: http://www.aptly.info/img/aptly_logo.png
    :target: http://www.aptly.info/

Documentation is available at `http://www.aptly.info/ <http://www.aptly.info/>`_. For support please use
open `issues <https://github.com/aptly-dev/aptly/issues>`_ or `discussions <https://github.com/aptly-dev/aptly/discussions>`_.

Aptly features:

* make mirrors of remote Debian/Ubuntu repositories, limiting by components/architectures
* take snapshots of mirrors at any point in time, fixing state of repository at some moment of time
* publish snapshot as Debian repository, ready to be consumed by apt
* controlled update of one or more packages in snapshot from upstream mirror, tracking dependencies
* merge two or more snapshots into one
* filter repository by search query, pulling dependencies when required
* publish self-made packages as Debian repositories
* REST API for remote access

Any contributions are welcome! Please see `CONTRIBUTING.md <CONTRIBUTING.md>`_.

Installation
=============

Aptly can be installed on several operating systems.

Debian / Ubuntu
----------------

Aptly is provided in the following debian packages:

* **aptly**: Includes the main Aptly binary, man pages, and shell completions
* **aptly-api**: A systemd service for the REST API, using the global /etc/aptly.conf
* **aptly-dbg**: Debug symbols for troubleshooting

The packages can be installed on official `Debian <https://packages.debian.org/search?keywords=aptly>`_ and `Ubuntu <https://packages.ubuntu.com/search?keywords=aptly>`_ distributions.

Upstream Debian Packages
~~~~~~~~~~~~~~~~~~~~~~~~~

If a newer version (not available in Debian/Ubuntu) of aptly is required, upstream debian packages (built from git tags) can be installed as follows:

Install the following APT key (as root)::

    wget -O /etc/apt/keyrings/aptly.asc https://www.aptly.info/pubkey.txt

Define Release APT sources in ``/etc/apt/sources.list.d/aptly.list``::

    deb [signed-by=/etc/apt/keyrings/aptly.asc] http://repo.aptly.info/release DIST main

Where DIST is one of: ``buster``, ``bullseye``, ``bookworm``, ``focal``, ``jammy``, ``noble``

Install aptly packages::

    apt-get update
    apt-get install aptly
    apt-get install aptly-api  # REST API systemd service

CI Builds
~~~~~~~~~~

For testing new features or bugfixes, recent builds are available as CI builds (built from master, may be unstable!) and can be installed as follows:

Define CI APT sources in ``/etc/apt/sources.list.d/aptly-ci.list``::

    deb [signed-by=/etc/apt/keyrings/aptly.asc] http://repo.aptly.info/ci DIST main

Where DIST is one of: ``buster``, ``bullseye``, ``bookworm``, ``focal``, ``jammy``, ``noble``

Note: same gpg key is used as for the Upstream Debian Packages.

Other Operating Systems
------------------------

Binary executables (depends almost only on libc) are available on `GitHub Releases <https://github.com/aptly-dev/aptly/releases>`_ for:

- macOS / darwin (amd64, arm64)
- FreeBSD (amd64, arm64, 386, arm)
- Generic Linux (amd64, arm64, 386, arm)

Integrations
=============

Vagrant:

-   `Vagrant configuration <https://github.com/sepulworld/aptly-vagrant>`_ by
    Zane Williamson, allowing to bring two virtual servers, one with aptly installed
    and another one set up to install packages from repository published by aptly

Docker:

-    `Docker container <https://github.com/mikepurvis/aptly-docker>`_ with aptly inside by Mike Purvis
-    `Docker container <https://github.com/urpylka/docker-aptly>`_ with aptly and nginx by Artem Smirnov

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

Molior:

-   `Molior Debian Build System <https://github.com/molior-dbs/molior>`_ by André Roth

Configuration
=============

etcd Database Configuration
---------------------------

When using etcd as the database backend, aptly supports several environment variables for configuration:

**Timeout Configuration:**

- ``APTLY_ETCD_TIMEOUT``: Operation timeout for etcd requests (default: ``60s``)
  
  Example: ``export APTLY_ETCD_TIMEOUT=30s``

- ``APTLY_ETCD_DIAL_TIMEOUT``: Connection timeout when establishing etcd connection (default: ``60s``)
  
  Example: ``export APTLY_ETCD_DIAL_TIMEOUT=10s``

**Connection Configuration:**

- ``APTLY_ETCD_KEEPALIVE``: Keep-alive timeout for etcd connections (default: ``7200s``)
  
  Example: ``export APTLY_ETCD_KEEPALIVE=3600s``

- ``APTLY_ETCD_MAX_MSG_SIZE``: Maximum message size in bytes for etcd requests/responses (default: ``52428800`` - 50MB)
  
  Example: ``export APTLY_ETCD_MAX_MSG_SIZE=104857600``  # 100MB

**Example Configuration:**

.. code-block:: bash

    # Set shorter timeouts for faster failure detection
    export APTLY_ETCD_TIMEOUT=30s
    export APTLY_ETCD_DIAL_TIMEOUT=10s
    
    # Increase message size for large package operations
    export APTLY_ETCD_MAX_MSG_SIZE=104857600
    
    # Run aptly with etcd backend
    aptly -config=/etc/aptly-etcd.conf mirror update debian-stable

**Features:**

- **Automatic Retry**: Read operations (Get) automatically retry up to 3 times with exponential backoff on temporary failures
- **Timeout Protection**: All etcd operations use context with timeout to prevent indefinite hangs
- **Enhanced Logging**: All etcd errors are logged with operation context for better debugging
- **Configurable Limits**: Message size limits can be adjusted for large package operations

etcd Write Queue Configuration
------------------------------

To prevent etcd overload during concurrent operations (e.g., multiple mirror updates), aptly supports an optional write queue that serializes database write operations:

**Configuration in aptly.conf:**

.. code-block:: json

    {
      "databaseBackend": {
        "type": "etcd",
        "url": "localhost:2379",
        "timeout": "120s",
        "writeRetries": 3,
        "writeQueue": {
          "enabled": true,
          "queueSize": 1000,
          "maxWritesPerSec": 100,
          "batchMaxSize": 50,
          "batchMaxWaitMs": 10
        }
      }
    }

**Write Queue Options:**

- ``enabled``: Enable/disable the write queue (default: ``false``)
- ``queueSize``: Size of the write operation queue (default: ``1000``)
- ``maxWritesPerSec``: Maximum write operations per second (default: ``100``)
- ``batchMaxSize``: Maximum batch size for future batching support (default: ``50``)
- ``batchMaxWaitMs``: Maximum wait time for batch accumulation in milliseconds (default: ``10``)

**Benefits:**

- **Prevents etcd Overload**: Serializes write operations to avoid overwhelming etcd
- **Maintains Parallelism**: I/O operations like downloads remain parallel
- **Rate Limiting**: Configurable writes per second to match etcd capacity
- **Transparent**: No code changes required, just enable in configuration

**Example Impact:**

Without write queue: 5 mirror updates → 5 parallel writers → 1000s of concurrent etcd operations → timeouts

With write queue: 5 mirror updates → 5 parallel processes → 1 sequential etcd writer → stable performance
