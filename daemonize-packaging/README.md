Aptly packaging with daemonize scripts

# Building package

To build the package, you have to:

 1. clone aptly github repository into ./src/github.com/smira/aptly
    This come from the existing packaging mechanisms.
 2. then run: $ dpkg-buildpackage -us -uc

# Service configuration

Service configuration will be located at /etc/aptly/aptlyd.conf
In that case, the modification of the daemon configuration will not impact the
global aptly.conf file.


# Included files

## debian/aptly.default
Contains default variable. It is placed into /etc/default/aptly

## debian/conffiles
Contain the aplty configuration file declaration in order to avoid it to be
removed when only removing aptly package

## debian/control
Add new dependencies

## debian/dirs
Declare the /var/aptly directory in order to create it during installation

## debian/init.d
The /etc/init.d/aptly script

## debian/install
where file has to be located during installation

## debian/post{inst|rm}
script to be run just after the installation and removal of the package.
It fixes rights and creates the aptlyuser

