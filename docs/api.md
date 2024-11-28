Aptly operations are also available via REST API served with `aptly api serve`.

On Debian based systems, a package `aptly-api` is available, which will run aptly as systemd service as dedicated aptly-api user.

Some configuration changes (S3 publishing endpoints, ...) will require restarting the aptly service in order to take effect.

The REST API shouldn't be exposed to the Internet as there is no authentication/protection, consider using a HTTP proxy (e.g. nginx) to add https and authentication.

#### Aptly REST API Documentation
