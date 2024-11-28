Aptly operations are also available via REST API served with `aptly api serve`.

On Debian based systems, a package `aptly-api` is available, which will run aptly as systemd service as dedicated aptly-api user.

#### Example API calls

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

#### Notes

- Some configuration changes (S3 publishing endpoints, ...) will require restarting the aptly service
- Aptly's HTTP API shouldn't be directly exposed to the Internet as there is no authentication/protection of APIs. Consider using a HTTP proxy or server (e.g. nginx) to add an authentication mechanism.

#### Aptly REST API Documentation
