Partial import of https://github.com/coreos/go-systemd to avoid a build dependency on systemd-dev for maximum build compatibility across different environments.

This import only includes activation code without tests as the tests use code from another directory making them not relocatable without introducing a delta to make them pass.

Code is Apache-2 which is equally permissive as MIT, which is used for aptly.
