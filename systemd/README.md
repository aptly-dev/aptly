Partial import of https://github.com/coreos/go-systemd to avoid a build dependency on systemd-dev (which is not reasonably available on the type of Travis CI that is used - i.e. Ubuntu 14.04).

This import only includes activation code without tests as the tests use code from another directory making them not relocatable without introducing a delta to make them pass.

Code is Apache-2 which is equally permissive as MIT, which is used for aptly.
