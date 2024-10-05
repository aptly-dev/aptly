
Using aptly via REST API allows to achieve two goals:

1. Remote access to aptly service: e.g. uploading packages and publishing them from CI server.
2. Concurrent access to aptly by multiple users.

#### Quickstart

Run `aptly api serve` to start HTTP service:

    $ aptly api serve
    Starting web server at: :8080 (press Ctrl+C to quit)...
    [GIN-debug] GET   /api/version              --> github.com/aptly-dev/aptly/api.apiVersion (4 handlers)
    ...

By default aptly would listen on `:8080`, but it could be changed with `-listen` flag.

Usage:

    $ aptly api serve -listen=:8080

Flags:

-   `-listen=":8080"`: host:port for HTTP listening
-   `-no-lock`: don't lock the database

When `-no-lock` option is enabled, API server acquires and drops the lock
around all the operations, so that API and CLI could be used concurrently.

Try some APIs:

    $ curl http://localhost:8080/api/version
    {"Version":"0.9~dev"}

    $ curl -F file=@aptly_0.8_i386.deb http://localhost:8080/api/files/aptly_0.8
    ["aptly_0.8/aptly_0.8_i386.deb"]

    $ curl -X POST -H 'Content-Type: application/json' --data '{"Name": "aptly-repo"}' http://localhost:8080/api/repos
    {"Name":"aptly-repo","Comment":"","DefaultDistribution":"","DefaultComponent":""}

    $ curl -X POST http://localhost:8080/api/repos/aptly-repo/file/aptly_0.8
    {"failedFiles":[],"report":{"warnings":[],"added":["aptly_0.8_i386 added"],"removed":[]}}

    $ curl http://localhost:8080/api/repos/aptly-repo/packages
    ["Pi386 aptly 0.8 966561016b44ed80"]

    $ curl -X POST -H 'Content-Type: application/json' --data '{"Distribution": "wheezy", "Sources": [{"Name": "aptly-repo"}]}' http://localhost:8080/api/publish//repos
    {"Architectures":["i386"],"Distribution":"wheezy","Label":"","Origin":"","Prefix":".","SourceKind":"local","Sources":[{"Component":"main","Name":"aptly-repo"}],"Storage":""}

#### Security

For security reasons it is advised to let Aptly listen on a Unix domain socket rather than a port. Sockets are subject to file permissions and thus allow for user-level access control while binding to a port only gives host-level access control. To use a socket simply run Aptly with a suitable listen flag such as `aptly api serve -listen=unix:///var/run/aptly.sock`.

Aptly's HTTP API shouldn't be directly exposed to the Internet: there's no authentication/protection of APIs. To publish the API it could be proxied through a HTTP proxy or server (e.g. nginx) to add an authentication mechanism or disallow all non-GET requests. [Reference example](https://github.com/sepich/nginx-ldap) for LDAP based per-repo access with nginx.

#### Notes

1. Some things (for example, S3 publishing endpoints) could be set up only using configuration file, so it requires restart of aptly HTTP server in order for changes to take effect.
1. GPG key passphrase can't be input on console, so either passwordless GPG keys are required or passphrase should be specified in API parameters.
1. Some script might be required to start/stop aptly HTTP service.
1. Some parameters are given as part of URLs, which requires proper url encoding. Unfortunately, at the moment it's not possible to pass URL arguments with `/` in them.

### How to implement equivalent of aptly commands using API

* `aptly mirror`: [mirror API](/doc/api/mirror)
    * `list`: list
    * `create`: create
    * `drop`: delete
    * `show`: show
    * `search`: show packages/search
    * `update`: update mirror
* `aptly repo`: [local repos API](/doc/api/repos)
    * `add`: [file upload API](/doc/api/files) + add packages from uploaded file
    * `copy`: show packages/search + add packages by key
    * `create`: create
    * `drop`: delete
    * `edit`: edit
    * `import`: not available, as mirror API is not implemented yet
    * `list`: list
    * `move`: show packages/search + add packages by key + delete packages by key
    * `remove`: show packages/search + delete packages by key
    * `rename`: not available yet, should be part of edit API
    * `search`: show packages/search
    * `show`: show
* `aptly snapshot`: [snapshots API](/doc/api/snapshots)
    * `create`:
        * empty: create snapshot with empty package references
        * from mirror: not available yet
        * from local repo: create snapshot from local repo
    * `diff`: snapshot difference API
    * `drop`: delete
    * `filter`: show packages/search + create snapshot from package references
    * `list`: list
    * `merge`: show packages/search + processing + create snapshot from package references
    * `pull`: show packages/search + processing + create snapshot from package references (might be implemented as API in future versions)
    * `rename`: edit
    * `search`: show packages/search
    * `show`: show
    * `verify`: not available yet
* `aptly publish`: [publish API](/doc/api/publish)
    * `drop`: delete
    * `list`: list
    * `repo`: publish repo
    * `snapshot`: publish snapshot
    * `switch`: switch/update
    * `update`: switch/update
* `aptly package`: [packages API](/doc/api/packages)
    * `search`: not available yet
    * `show`: only one format, with package key as input
* `aptly graph`: [graph API](/doc/api/misc)
* `aptly version`: [version API](/doc/api/misc)
* `aptly db`: not available yet
